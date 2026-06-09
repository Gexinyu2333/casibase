// Copyright 2025 The OpenAgent Authors. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package office

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
)

const (
	nsDrawingML     = "http://schemas.openxmlformats.org/drawingml/2006/main"
	nsChart         = "http://schemas.openxmlformats.org/drawingml/2006/chart"
	nsPresentation  = "http://schemas.openxmlformats.org/presentationml/2006/main"
	nsOfficeRels    = "http://schemas.openxmlformats.org/officeDocument/2006/relationships"
	nsSpreadsheetML = "http://schemas.openxmlformats.org/spreadsheetml/2006/main"
	nsP14           = "http://schemas.microsoft.com/office/powerpoint/2010/main"
)

type xmlNode struct {
	Name     xml.Name
	Attr     []xml.Attr
	Children []*xmlNode
	Text     string
}

func parseXML(data []byte) (*xmlNode, error) {
	decoder := xml.NewDecoder(bytes.NewReader(data))
	var root *xmlNode
	var stack []*xmlNode
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		switch item := token.(type) {
		case xml.StartElement:
			node := &xmlNode{Name: item.Name, Attr: append([]xml.Attr(nil), item.Attr...)}
			if len(stack) == 0 {
				if root != nil {
					return nil, fmt.Errorf("XML has multiple roots")
				}
				root = node
			} else {
				stack[len(stack)-1].Children = append(stack[len(stack)-1].Children, node)
			}
			stack = append(stack, node)
		case xml.EndElement:
			if len(stack) == 0 {
				return nil, fmt.Errorf("unexpected XML end element")
			}
			stack = stack[:len(stack)-1]
		case xml.CharData:
			if len(stack) != 0 {
				stack[len(stack)-1].Text += string(item)
			}
		}
	}
	if root == nil {
		return nil, fmt.Errorf("XML document is empty")
	}
	return root, nil
}

func marshalXML(root *xmlNode) ([]byte, error) {
	var buffer bytes.Buffer
	buffer.WriteString(xml.Header)
	context := map[string]string{
		"xml": "http://www.w3.org/XML/1998/namespace",
	}
	if err := encodeXMLNode(&buffer, root, context, new(int)); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

type namespaceDeclaration struct {
	Prefix string
	URI    string
}

var preferredNamespacePrefixes = map[string]string{
	nsDrawingML:     "a",
	nsChart:         "c",
	nsPresentation:  "p",
	nsOfficeRels:    "r",
	nsSpreadsheetML: "x",
	nsP14:           "p14",
}

func encodeXMLNode(buffer *bytes.Buffer, node *xmlNode, parent map[string]string, counter *int) error {
	context := cloneNamespaceContext(parent)
	var declarations []namespaceDeclaration
	var attrs []xml.Attr
	for _, attr := range node.Attr {
		prefix, ok := namespaceAttribute(attr)
		if !ok {
			attrs = append(attrs, attr)
			continue
		}
		context[prefix] = attr.Value
		declarations = append(declarations, namespaceDeclaration{Prefix: prefix, URI: attr.Value})
	}

	elementPrefix := ensureNamespacePrefix(context, node.Name.Space, true, counter, &declarations)
	attrPrefixes := make([]string, len(attrs))
	for index, attr := range attrs {
		if attr.Name.Space != "" {
			attrPrefixes[index] = ensureNamespacePrefix(context, attr.Name.Space, false, counter, &declarations)
		}
	}

	buffer.WriteByte('<')
	writeQualifiedName(buffer, elementPrefix, node.Name.Local)
	for _, declaration := range declarations {
		buffer.WriteString(` xmlns`)
		if declaration.Prefix != "" {
			buffer.WriteByte(':')
			buffer.WriteString(declaration.Prefix)
		}
		buffer.WriteString(`="`)
		if err := xml.EscapeText(buffer, []byte(declaration.URI)); err != nil {
			return err
		}
		buffer.WriteByte('"')
	}
	for index, attr := range attrs {
		buffer.WriteByte(' ')
		writeQualifiedName(buffer, attrPrefixes[index], attr.Name.Local)
		buffer.WriteString(`="`)
		if err := xml.EscapeText(buffer, []byte(attr.Value)); err != nil {
			return err
		}
		buffer.WriteByte('"')
	}
	buffer.WriteByte('>')

	if node.Text != "" {
		if err := xml.EscapeText(buffer, []byte(node.Text)); err != nil {
			return err
		}
	}
	for _, child := range node.Children {
		if err := encodeXMLNode(buffer, child, context, counter); err != nil {
			return err
		}
	}
	buffer.WriteString("</")
	writeQualifiedName(buffer, elementPrefix, node.Name.Local)
	buffer.WriteByte('>')
	return nil
}

func namespaceAttribute(attr xml.Attr) (string, bool) {
	if attr.Name.Space == "" && attr.Name.Local == "xmlns" {
		return "", true
	}
	if attr.Name.Space == "xmlns" {
		return attr.Name.Local, true
	}
	return "", false
}

func cloneNamespaceContext(source map[string]string) map[string]string {
	result := make(map[string]string, len(source))
	for prefix, uri := range source {
		result[prefix] = uri
	}
	return result
}

func ensureNamespacePrefix(context map[string]string, uri string, allowDefault bool, counter *int, declarations *[]namespaceDeclaration) string {
	if uri == "" {
		return ""
	}
	if prefix := existingNamespacePrefix(context, uri, allowDefault); prefix != "" || (allowDefault && context[""] == uri) {
		return prefix
	}
	prefix := preferredNamespacePrefixes[uri]
	if prefix == "" || (context[prefix] != "" && context[prefix] != uri) {
		for {
			*counter++
			prefix = fmt.Sprintf("ns%d", *counter)
			if _, exists := context[prefix]; !exists {
				break
			}
		}
	}
	context[prefix] = uri
	*declarations = append(*declarations, namespaceDeclaration{Prefix: prefix, URI: uri})
	return prefix
}

func existingNamespacePrefix(context map[string]string, uri string, allowDefault bool) string {
	if preferred := preferredNamespacePrefixes[uri]; preferred != "" && context[preferred] == uri {
		return preferred
	}
	if allowDefault && context[""] == uri {
		return ""
	}
	var prefixes []string
	for prefix, boundURI := range context {
		if prefix != "" && boundURI == uri {
			prefixes = append(prefixes, prefix)
		}
	}
	sort.Strings(prefixes)
	if len(prefixes) != 0 {
		return prefixes[0]
	}
	return ""
}

func writeQualifiedName(buffer *bytes.Buffer, prefix, local string) {
	if prefix != "" {
		buffer.WriteString(prefix)
		buffer.WriteByte(':')
	}
	buffer.WriteString(local)
}

func (n *xmlNode) clone() *xmlNode {
	if n == nil {
		return nil
	}
	result := &xmlNode{Name: n.Name, Attr: append([]xml.Attr(nil), n.Attr...), Text: n.Text}
	result.Children = make([]*xmlNode, len(n.Children))
	for index, child := range n.Children {
		result.Children[index] = child.clone()
	}
	return result
}

func (n *xmlNode) attr(space, local string) string {
	for _, item := range n.Attr {
		if item.Name.Space == space && item.Name.Local == local {
			return item.Value
		}
	}
	return ""
}

func (n *xmlNode) setAttr(space, local, value string) {
	for index := range n.Attr {
		if n.Attr[index].Name.Space == space && n.Attr[index].Name.Local == local {
			n.Attr[index].Value = value
			return
		}
	}
	n.Attr = append(n.Attr, xml.Attr{Name: xml.Name{Space: space, Local: local}, Value: value})
}

func (n *xmlNode) child(space, local string) *xmlNode {
	for _, child := range n.Children {
		if child.Name.Space == space && child.Name.Local == local {
			return child
		}
	}
	return nil
}

func (n *xmlNode) children(space, local string) []*xmlNode {
	var result []*xmlNode
	for _, child := range n.Children {
		if child.Name.Space == space && child.Name.Local == local {
			result = append(result, child)
		}
	}
	return result
}

func (n *xmlNode) descendants(space, local string) []*xmlNode {
	var result []*xmlNode
	var walk func(*xmlNode)
	walk = func(current *xmlNode) {
		for _, child := range current.Children {
			if child.Name.Space == space && child.Name.Local == local {
				result = append(result, child)
			}
			walk(child)
		}
	}
	walk(n)
	return result
}

func (n *xmlNode) firstDescendant(space, local string) *xmlNode {
	for _, child := range n.Children {
		if child.Name.Space == space && child.Name.Local == local {
			return child
		}
		if found := child.firstDescendant(space, local); found != nil {
			return found
		}
	}
	return nil
}

func (n *xmlNode) removeChildren(space, local string) {
	kept := n.Children[:0]
	for _, child := range n.Children {
		if child.Name.Space != space || child.Name.Local != local {
			kept = append(kept, child)
		}
	}
	n.Children = kept
}

func element(space, local string, attrs ...xml.Attr) *xmlNode {
	return &xmlNode{Name: xml.Name{Space: space, Local: local}, Attr: attrs}
}

func plainAttr(name, value string) xml.Attr {
	return xml.Attr{Name: xml.Name{Local: name}, Value: value}
}

func intAttr(node *xmlNode, space, name string) (int, bool) {
	raw := node.attr(space, name)
	value, err := strconv.Atoi(raw)
	return value, err == nil
}

func floatAttr(node *xmlNode, space, name string) (float64, bool) {
	raw := node.attr(space, name)
	value, err := strconv.ParseFloat(raw, 64)
	return value, err == nil
}

func textContent(node *xmlNode) string {
	if node == nil {
		return ""
	}
	var builder strings.Builder
	builder.WriteString(node.Text)
	for _, child := range node.Children {
		builder.WriteString(textContent(child))
	}
	return builder.String()
}
