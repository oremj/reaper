// generated by stringer -type=ResourceState; DO NOT EDIT

package main

import "fmt"

const _ResourceState_name = "pendingrunningshuttingDownterminatedstoppingstopped"

var _ResourceState_index = [...]uint8{0, 7, 14, 26, 36, 44, 51}

func (i ResourceState) String() string {
	if i < 0 || i >= ResourceState(len(_ResourceState_index)-1) {
		return fmt.Sprintf("ResourceState(%d)", i)
	}
	return _ResourceState_name[_ResourceState_index[i]:_ResourceState_index[i+1]]
}