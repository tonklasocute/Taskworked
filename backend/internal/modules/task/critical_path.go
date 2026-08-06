package task

import (
	"errors"
	"sort"
	"time"

	"github.com/google/uuid"
)

var ErrDependencyCycle = errors.New("dependency graph contains a cycle")

type cpmNode struct {
	id       uuid.UUID
	start    time.Time // the task's own recorded start (anchor for source nodes)
	duration int       // days; 0 for a milestone (start == due)
}

// computeCriticalPath runs the Critical Path Method (forward pass for
// earliest start/finish, backward pass for latest start/finish, slack =
// latest - earliest) over tasks connected by finish-to-start dependencies.
//
// Tasks without a DueDate are excluded — there's nothing to schedule.
// Dependencies referencing an excluded or unknown task are ignored rather
// than erroring, since a task's own missing dates is either a legitimate
// backlog task or something the UI already flags, and shouldn't take the
// dependency graph down with it.
//
// Returns the critical path as task IDs in source-to-sink order, plus the
// total project duration in days (0 if there's nothing to schedule).
// Returns ErrDependencyCycle if the schedulable subgraph has a cycle.
func computeCriticalPath(tasks []Task, deps []Dependency) ([]uuid.UUID, int, error) {
	nodes := buildNodes(tasks)
	if len(nodes) == 0 {
		return nil, 0, nil
	}

	predecessors, successors := buildEdges(nodes, deps)

	order, err := topologicalOrder(nodes, predecessors)
	if err != nil {
		return nil, 0, err
	}

	es, ef := forwardPass(order, nodes, predecessors)

	projectFinish := ef[order[0]]
	for _, id := range order {
		if ef[id].After(projectFinish) {
			projectFinish = ef[id]
		}
	}

	ls, _ := backwardPass(order, nodes, successors, projectFinish)

	path := reconstructPath(order, nodes, successors, es, ef, ls)

	totalDays := 0
	if len(path) > 0 {
		totalDays = daysBetween(es[path[0]], ef[path[len(path)-1]])
	}

	return path, totalDays, nil
}

func buildNodes(tasks []Task) map[uuid.UUID]cpmNode {
	nodes := make(map[uuid.UUID]cpmNode, len(tasks))
	for _, t := range tasks {
		if t.DueDate == nil {
			continue
		}
		start := *t.DueDate
		if t.StartDate != nil {
			start = *t.StartDate
		}
		duration := daysBetween(start, *t.DueDate)
		if duration < 0 {
			duration = 0
		}
		nodes[t.ID] = cpmNode{id: t.ID, start: start, duration: duration}
	}
	return nodes
}

func buildEdges(nodes map[uuid.UUID]cpmNode, deps []Dependency) (predecessors, successors map[uuid.UUID][]uuid.UUID) {
	predecessors = make(map[uuid.UUID][]uuid.UUID)
	successors = make(map[uuid.UUID][]uuid.UUID)
	for _, d := range deps {
		if _, ok := nodes[d.TaskID]; !ok {
			continue
		}
		if _, ok := nodes[d.DependsOnID]; !ok {
			continue
		}
		predecessors[d.TaskID] = append(predecessors[d.TaskID], d.DependsOnID)
		successors[d.DependsOnID] = append(successors[d.DependsOnID], d.TaskID)
	}
	return predecessors, successors
}

// topologicalOrder runs Kahn's algorithm restricted to the given node set.
// Deterministic: among tasks with equal in-degree, the one with the
// earlier recorded start (then lower ID) is emitted first.
func topologicalOrder(nodes map[uuid.UUID]cpmNode, predecessors map[uuid.UUID][]uuid.UUID) ([]uuid.UUID, error) {
	inDegree := make(map[uuid.UUID]int, len(nodes))
	for id := range nodes {
		inDegree[id] = len(predecessors[id])
	}

	successorsOf := make(map[uuid.UUID][]uuid.UUID)
	for id, preds := range predecessors {
		for _, p := range preds {
			successorsOf[p] = append(successorsOf[p], id)
		}
	}

	var ready []uuid.UUID
	for id, deg := range inDegree {
		if deg == 0 {
			ready = append(ready, id)
		}
	}

	order := make([]uuid.UUID, 0, len(nodes))
	for len(ready) > 0 {
		sortNodeIDs(ready, nodes)
		next := ready[0]
		ready = ready[1:]
		order = append(order, next)

		for _, succ := range successorsOf[next] {
			inDegree[succ]--
			if inDegree[succ] == 0 {
				ready = append(ready, succ)
			}
		}
	}

	if len(order) != len(nodes) {
		return nil, ErrDependencyCycle
	}
	return order, nil
}

func sortNodeIDs(ids []uuid.UUID, nodes map[uuid.UUID]cpmNode) {
	sort.Slice(ids, func(i, j int) bool {
		a, b := nodes[ids[i]], nodes[ids[j]]
		if !a.start.Equal(b.start) {
			return a.start.Before(b.start)
		}
		return a.id.String() < b.id.String()
	})
}

func forwardPass(order []uuid.UUID, nodes map[uuid.UUID]cpmNode, predecessors map[uuid.UUID][]uuid.UUID) (es, ef map[uuid.UUID]time.Time) {
	es = make(map[uuid.UUID]time.Time, len(order))
	ef = make(map[uuid.UUID]time.Time, len(order))

	for _, id := range order {
		node := nodes[id]
		start := node.start
		for _, pred := range predecessors[id] {
			if ef[pred].After(start) {
				start = ef[pred]
			}
		}
		es[id] = start
		ef[id] = start.AddDate(0, 0, node.duration)
	}
	return es, ef
}

func backwardPass(order []uuid.UUID, nodes map[uuid.UUID]cpmNode, successors map[uuid.UUID][]uuid.UUID, projectFinish time.Time) (ls, lf map[uuid.UUID]time.Time) {
	ls = make(map[uuid.UUID]time.Time, len(order))
	lf = make(map[uuid.UUID]time.Time, len(order))

	for i := len(order) - 1; i >= 0; i-- {
		id := order[i]
		node := nodes[id]
		finish := projectFinish
		if succs := successors[id]; len(succs) > 0 {
			finish = ls[succs[0]]
			for _, succ := range succs[1:] {
				if ls[succ].Before(finish) {
					finish = ls[succ]
				}
			}
		}
		lf[id] = finish
		ls[id] = finish.AddDate(0, 0, -node.duration)
	}
	return ls, lf
}

// reconstructPath walks from a zero-slack task with no zero-slack
// predecessor, forward through zero-slack successors, always preferring
// the successor with the longest remaining chain so the reported path is
// a genuine longest path rather than an arbitrary critical branch.
func reconstructPath(order []uuid.UUID, nodes map[uuid.UUID]cpmNode, successors map[uuid.UUID][]uuid.UUID, es, ef, ls map[uuid.UUID]time.Time) []uuid.UUID {
	isCritical := func(id uuid.UUID) bool {
		return daysBetween(es[id], ls[id]) == 0
	}

	// Longest remaining critical chain starting at each node, computed by
	// walking the (already topologically sorted) order in reverse.
	chainLen := make(map[uuid.UUID]int, len(order))
	for i := len(order) - 1; i >= 0; i-- {
		id := order[i]
		if !isCritical(id) {
			continue
		}
		best := 0
		for _, succ := range successors[id] {
			if isCritical(succ) && ef[id].Equal(es[succ]) && chainLen[succ] > best {
				best = chainLen[succ]
			}
		}
		chainLen[id] = nodes[id].duration + best
	}

	var start uuid.UUID
	started := false
	for _, id := range order {
		if !isCritical(id) {
			continue
		}
		if !started || chainLen[id] > chainLen[start] {
			start = id
			started = true
		}
	}
	if !started {
		return nil
	}

	path := []uuid.UUID{start}
	current := start
	for {
		var next uuid.UUID
		found := false
		for _, succ := range successors[current] {
			if isCritical(succ) && ef[current].Equal(es[succ]) {
				if !found || chainLen[succ] > chainLen[next] {
					next = succ
					found = true
				}
			}
		}
		if !found {
			break
		}
		path = append(path, next)
		current = next
	}
	return path
}

func daysBetween(a, b time.Time) int {
	return int(b.Sub(a).Hours() / 24)
}

// canReach reports whether to is reachable from "from" by following
// successor edges (predecessor -> successor). Used to reject a new
// dependency that would close a cycle: adding "taskID depends on
// dependsOnID" is only safe if taskID cannot already reach dependsOnID —
// otherwise dependsOnID -> ... -> taskID -> dependsOnID would loop.
func canReach(successors map[uuid.UUID][]uuid.UUID, from, to uuid.UUID) bool {
	visited := make(map[uuid.UUID]bool)
	var dfs func(uuid.UUID) bool
	dfs = func(node uuid.UUID) bool {
		if node == to {
			return true
		}
		if visited[node] {
			return false
		}
		visited[node] = true
		for _, next := range successors[node] {
			if dfs(next) {
				return true
			}
		}
		return false
	}
	return dfs(from)
}
