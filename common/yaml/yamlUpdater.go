package yaml

import (
	"Goauld/common/utils"
	"bytes"
	"os"

	"github.com/goccy/go-yaml"
	"github.com/goccy/go-yaml/ast"
	"github.com/goccy/go-yaml/parser"
	"github.com/goccy/go-yaml/printer"
)

// UpdateAgentPasswordConfig Updates the configuration file with the agent password.
func UpdateAgentPasswordConfig(file string, name, pass string) error {
	//nolint:gosec
	data, err := os.ReadFile(file)
	if err != nil {
		return err
	}
	content, err := parser.ParseBytes(data, parser.ParseComments)
	if err != nil {
		return err
	}
	node, err := yaml.PathString("$.agent-password")
	if err != nil {
		return err
	}

	curMap := make(map[string]string)
	_ = node.Read(bytes.NewReader(data), &curMap)
	// if err != nil {
	// We skip as we wil create the nod if it does not eststs
	// }
	if curMap[name] == pass {
		return nil
	}

	doc := content.Docs[0]
	root := doc.Body

	agentPassword := make(map[string]string)

	agentPassword[name] = pass

	exists, err := node.FilterNode(root)

	var apBytes []byte
	if err != nil || exists == nil || exists.Type() == ast.NullType {
		// The agent-password entry does not exist, so we will merge
		nodeParent, err := yaml.PathString("$")
		if err != nil {
			return err
		}
		node = nodeParent
		parentAgentPassword := make(map[string]map[string]string)
		parentAgentPassword["agent-password"] = agentPassword
		apBytes, err = yaml.Marshal(parentAgentPassword)
		if err != nil {
			return err
		}
	} else {
		apBytes, err = yaml.Marshal(agentPassword)
		if err != nil {
			return err
		}
	}
	n, err := parser.ParseBytes(apBytes, 0)
	if err != nil {
		return err
	}

	err = node.MergeFromReader(content, n)

	if err != nil {
		return err
	}

	p := printer.Printer{}
	result := p.PrintNode(content.Docs[0])

	return utils.OverwriteFile(file, result)
}
