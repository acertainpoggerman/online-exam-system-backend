package update

import (
	"fmt"
	"strings"
)

type Operation string

const (
	OperationAdd     Operation = "add"
	OperationRemove  Operation = "remove"
	OperationReplace Operation = "replace"
	OperationMove    Operation = "move"
	OperationCopy    Operation = "copy"
	OperationTest    Operation = "test"
)

type UpdateAction struct {
	Operation `json:"op"`
	Path      string `json:"path"`
	Value     any    `json:"value,omitempty"`
	From      string `json:"from,omitempty"`
}

type UpdateActionHandler func(UpdateAction, map[string]string) error
type OperationMap map[Operation]UpdateActionHandler
type RoutingMap map[string]OperationMap

// type UpdateMux struct{}

// func NewUpdateMux() *UpdateMux {
// 	return &UpdateMux{}
// }

func MatchesPattern(path string, pattern string) (map[string]string, bool) {
	partsPattern := strings.Split(strings.Trim(pattern, "/"), "/")
	partsPath := strings.Split(strings.Trim(path, "/"), "/")

	if len(partsPath) != len(partsPattern) {
		return nil, false
	}

	params := map[string]string{}
	for idx, pattern := range partsPattern {
		if strings.HasPrefix(pattern, "{") && strings.HasSuffix(pattern, "}") {
			key := pattern[1 : len(pattern)-1]
			params[key] = partsPath[idx]
		} else if pattern != partsPath[idx] {
			return nil, false
		}
	}

	return params, true
}

// Executes the last action that matches { op, pattern }.
// Will return nil if a found action was executed successfully, or
// if no actions were found.
//
// Will return an error if handleFunc returns an error.
func ExecuteLastInstanceOfAction(actions []UpdateAction, op Operation, pattern string, handleFunc UpdateActionHandler) error {

	if handleFunc == nil {
		return fmt.Errorf("Nil handler provided")
	}

	exists := false
	var finalAction UpdateAction
	params := map[string]string{}

	for _, action := range actions {
		if p, ok := MatchesPattern(action.Path, pattern); ok && op == action.Operation {
			exists = true
			finalAction = action
			params = p
		}
	}

	if exists {
		return handleFunc(finalAction, params)
	}

	return nil
}

func ExecuteAction(action UpdateAction, routes RoutingMap) error {

	for pattern, val := range routes {
		for op, handleFunc := range val {
			if params, matches := MatchesPattern(action.Path, pattern); matches && op == action.Operation {
				if handleFunc == nil {
					return fmt.Errorf("Nil handler provided for route { op: \"%v\" path: \"%v\" }", op, pattern)
				}
				return handleFunc(action, params)
			}
		}
	}

	return nil
}
