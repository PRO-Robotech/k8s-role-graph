package engine

import (
	"k8s-role-graph/internal/indexer"
	api "k8s-role-graph/pkg/apis/rbacgraph/v1alpha1"
)

func (qc *queryContext) buildRBACGraph(roleIDs []indexer.RoleID) {
	e := &Engine{}
	for _, roleID := range roleIDs {
		role, ok := qc.snapshot.RolesByID[roleID]
		if !ok {
			continue
		}
		if !allowNamespace(qc.namespaceFilter, role.Namespace, false) {
			continue
		}

		matches := e.matchRole(role, qc.spec)
		if len(matches) == 0 {
			continue
		}

		roleRefKey := indexer.RoleRefKey{Kind: role.Kind, Namespace: role.Namespace, Name: role.Name}
		bindings := qc.snapshot.BindingsByRoleRef[roleRefKey]
		filteredBindings := filterBindingsByNamespace(qc.namespaceFilter, qc.namespaceStrict, bindings)
		if qc.namespaceStrict && role.Namespace == "" && len(filteredBindings) == 0 {
			continue
		}

		roleNodeID := upsertRoleNode(&qc.status.Graph.Nodes, qc.nodeSeen, qc.nodeIndex, role, qc.snapshot.AggregatedRoleSources[roleID], matches)
		qc.roleSeen[roleID] = struct{}{}
		for _, sourceRoleID := range qc.snapshot.AggregatedRoleSources[roleID] {
			sourceRole, ok := qc.snapshot.RolesByID[sourceRoleID]
			if !ok {
				continue
			}
			sourceNodeID := upsertRoleNode(&qc.status.Graph.Nodes, qc.nodeSeen, qc.nodeIndex, sourceRole, qc.snapshot.AggregatedRoleSources[sourceRoleID], nil)
			edgeID := edgeIDFor(sourceNodeID, roleNodeID, api.GraphEdgeTypeAggregates)
			if _, exists := qc.edgeSeen[edgeID]; !exists {
				qc.status.Graph.Edges = append(qc.status.Graph.Edges, api.GraphEdge{
					ID:      edgeID,
					From:    sourceNodeID,
					To:      roleNodeID,
					Type:    api.GraphEdgeTypeAggregates,
					Explain: edgeExplainAggregates,
				})
				qc.edgeSeen[edgeID] = struct{}{}
			}
		}

		if len(filteredBindings) == 0 {
			accumulateResourceRows(qc.resourceRows, matches, roleID, "", "")
			continue
		}

		for _, binding := range filteredBindings {
			bindingNodeIDValue := bindingNodeID(binding)
			if _, exists := qc.nodeSeen[bindingNodeIDValue]; !exists {
				qc.status.Graph.Nodes = append(qc.status.Graph.Nodes, api.GraphNode{
					ID:        bindingNodeIDValue,
					Type:      bindingType(binding),
					Name:      binding.Name,
					Namespace: binding.Namespace,
				})
				qc.nodeSeen[bindingNodeIDValue] = struct{}{}
			}
			qc.bindingSeen[bindingNodeIDValue] = struct{}{}

			edgeID := edgeIDFor(roleNodeID, bindingNodeIDValue, api.GraphEdgeTypeGrants)
			if _, exists := qc.edgeSeen[edgeID]; !exists {
				qc.status.Graph.Edges = append(qc.status.Graph.Edges, api.GraphEdge{
					ID:       edgeID,
					From:     roleNodeID,
					To:       bindingNodeIDValue,
					Type:     api.GraphEdgeTypeGrants,
					RuleRefs: matches,
					Explain:  edgeExplainGrants,
				})
				qc.edgeSeen[edgeID] = struct{}{}
			}

			if len(binding.Subjects) == 0 {
				accumulateResourceRows(qc.resourceRows, matches, roleID, bindingNodeIDValue, "")
				continue
			}

			for _, subject := range binding.Subjects {
				subjectNodeIDValue := subjectNodeID(subject)
				if _, exists := qc.nodeSeen[subjectNodeIDValue]; !exists {
					qc.status.Graph.Nodes = append(qc.status.Graph.Nodes, api.GraphNode{
						ID:        subjectNodeIDValue,
						Type:      subjectType(subject.Kind),
						Name:      subject.Name,
						Namespace: subject.Namespace,
					})
					qc.nodeSeen[subjectNodeIDValue] = struct{}{}
				}
				qc.subjectSeen[subjectNodeIDValue] = struct{}{}
				trackServiceAccountSubject(qc.saSubjects, subjectNodeIDValue, subject, binding.Namespace)

				edgeID = edgeIDFor(bindingNodeIDValue, subjectNodeIDValue, api.GraphEdgeTypeSubjects)
				if _, exists := qc.edgeSeen[edgeID]; !exists {
					qc.status.Graph.Edges = append(qc.status.Graph.Edges, api.GraphEdge{
						ID:      edgeID,
						From:    bindingNodeIDValue,
						To:      subjectNodeIDValue,
						Type:    api.GraphEdgeTypeSubjects,
						Explain: edgeExplainSubjects,
					})
					qc.edgeSeen[edgeID] = struct{}{}
				}

				accumulateResourceRows(qc.resourceRows, matches, roleID, bindingNodeIDValue, subjectNodeIDValue)
			}
		}
	}
}
