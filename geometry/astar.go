// code of this file is a strongly modified version of code from
// github.com/beefsack/go-astar, which has the following license:
//
// Copyright (c) 2014 Michael Charles Alexander
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package geometry

func (pr *PathRange) initAstar() {
	if pr.AstarNodes == nil {
		pr.AstarNodes = &nodeMap{}
		max := pr.Rg.Size()
		pr.AstarNodes.Nodes = make([]node, max.X*max.Y)
		pr.AstarQueue = make(priorityQueue, 0, max.X*max.Y)
	}
}

func checkNodesIdx(nm *nodeMap) {
	if nm.Idx+1 > 0 {
		return
	}
	for i, n := range nm.Nodes {
		idx := 0
		if n.Idx == nm.Idx {
			idx = 1
		}
		n.Idx = idx
		nm.Nodes[i] = n
	}
	nm.Idx = 1
}
