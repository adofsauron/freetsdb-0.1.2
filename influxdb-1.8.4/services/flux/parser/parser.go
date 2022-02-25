package parser

import (
	"influxdb.cluster/services/flux/ast"
	fastparser "influxdb.cluster/services/flux/internal/parser"
)

// NewAST parses Flux query and produces an ast.Program
func NewAST(flux string) (*ast.Program, error) {
	return fastparser.NewAST([]byte(flux)), nil
}
