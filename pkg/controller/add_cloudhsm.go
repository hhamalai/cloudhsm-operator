package controller

import (
	"github.com/hhamalai/cloudhsm-operator/pkg/controller/cloudhsm"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, cloudhsm.Add)
}
