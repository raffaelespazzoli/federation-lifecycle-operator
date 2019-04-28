package controller

import "github.com/raffaelespazzoli/federation-lifecycle-operator/pkg/controller/multiplenamespacefederation"

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, multiplenamespacefederation.Add)
}
