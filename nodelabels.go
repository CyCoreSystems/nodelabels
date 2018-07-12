package nodelabels

import (
	"context"
	"log"

	"github.com/ericchiang/k8s"
	corev1 "github.com/ericchiang/k8s/apis/core/v1"
	"github.com/pkg/errors"
)

// Manager provides
type Manager interface {
	Watch(ctx context.Context, sig chan struct{}) error

	List(ctx context.Context) ([]*corev1.Node, error)

	Reconcile(ctx context.Context, desiredCount int) error
}

type kubeManager struct {
	kc *k8s.Client

	key string
	val string
}

// Watch waits for changes to kubernetes nodes, signaling on the provided channel when one of the nodes changes
func (m *kubeManager) Watch(ctx context.Context, sig chan struct{}) error {
	resourceModel := new(corev1.Node)
	w, err := m.kc.Watch(ctx, "", resourceModel)
	if err != nil {
		return errors.Wrap(err, "failed to watch nodes")
	}
	defer w.Close() // nolint

	for {
		ref := new(corev1.Node)
		if _, err = w.Next(ref); err != nil {
			return errors.Wrap(err, "error during watch of nodes")
		}

		select {
		case sig <- struct{}{}:
		default:
		}
	}
}

// List returns the current list of proxy nodes
func (m *kubeManager) List(ctx context.Context) (ret []*corev1.Node, err error) {
	list, err := m.listAllNodes(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get list of nodes")
	}

	return m.filterProxies(list), nil
}

func (m *kubeManager) listAllNodes(ctx context.Context) (ret []*corev1.Node, err error) {
	list := new(corev1.NodeList)
	if err = m.kc.List(ctx, "", list); err != nil {
		return
	}

	ret = append(ret, list.GetItems()...)
	return
}

func (m *kubeManager) filterProxies(list []*corev1.Node) (ret []*corev1.Node) {
	for _, n := range list {
		if m.isMatchingNode(n) {
			ret = append(ret, n)
		}
	}
	return
}

// Reconcile adds or removes a node, as and if necessary to match the desides node count
func (m *kubeManager) Reconcile(ctx context.Context, desiredCount int) (err error) {
	list, err := m.listAllNodes(ctx)
	if err != nil {
		return err
	}

	currentCount := len(m.filterProxies(list))
	switch {
	case currentCount < desiredCount:
		err = m.addNode(ctx, list)
		return errors.Wrapf(err, "failed to add additional node; current(%d) desired(%d)", currentCount, desiredCount)
	case currentCount > desiredCount:
		err = m.removeNode(ctx, list)
		return errors.Wrapf(err, "failed to remove node; current(%d) desired(%d)", currentCount, desiredCount)
	default:
		return nil
	}
}

func (m *kubeManager) isMatchingNode(n *corev1.Node) bool {
	l := n.GetMetadata().GetLabels()
	if sip, ok := l[m.key]; ok {
		if sip == m.val {
			return true
		}
	}
	return false
}

func (m *kubeManager) nodeIsAvailable(n *corev1.Node) bool {
	l := n.GetMetadata().GetLabels()
	if _, ok := l[m.key]; ok {
		return false
	}
	return true
}

func (m *kubeManager) addNode(ctx context.Context, list []*corev1.Node) error {
	for _, n := range list {
		if !m.nodeIsAvailable(n) {
			continue
		}

		n.Metadata.Labels[m.key] = m.val

		if err := m.kc.Update(ctx, n); err != nil {
			log.Printf("failed to assign node %v: %v", n.Metadata.Name, err)
			continue
		}

		return nil
	}
	return errors.New("no node assignable")
}

func (m *kubeManager) removeNode(ctx context.Context, list []*corev1.Node) error {
	for _, n := range list {
		if m.isMatchingNode(n) {
			delete(n.Metadata.Labels, m.key)

			if err := m.kc.Update(ctx, n); err != nil {
				log.Printf("failed to unassign node %v: %v", n.Metadata.Name, err)
				continue
			}
			return nil
		}
	}
	return errors.New("failed to find removable node")
}

// NewManager returns a new node manager, providing high-level control over a set of node labels
func NewManager(kc *k8s.Client, key, val string) Manager {
	return &kubeManager{
		kc:  kc,
		key: key,
		val: val,
	}
}
