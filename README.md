# NodeLabels

NodeLabels is a narrow-scoped tool which manages the number of kubernetes nodes
which have the configured label (key + value).  Any node which already has a
label of the same key will be ignored.  The number of nodes with the requested
label will be increased or decreased as necessary to match the desired count.

There is a binary in `cmd/nodelabels` and a slightly more generic library in the
root.


