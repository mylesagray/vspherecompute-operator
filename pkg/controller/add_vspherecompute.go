package controller

import (
	"github.com/mylesgray/vspherecompute/pkg/controller/vspherecompute"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, vspherecompute.Add)
}
