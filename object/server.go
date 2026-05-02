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

package object

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ThinkInAIXYZ/go-mcp/protocol"
	"github.com/the-open-agent/openagent/agent"
	"github.com/the-open-agent/openagent/i18n"
	mcppkg "github.com/the-open-agent/openagent/mcp"
	"github.com/the-open-agent/openagent/util"
	"xorm.io/core"
)

type McpTool struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	IsAllowed   bool   `json:"isAllowed"`
}

type Server struct {
	Owner       string `xorm:"varchar(100) notnull pk" json:"owner"`
	Name        string `xorm:"varchar(100) notnull pk" json:"name"`
	CreatedTime string `xorm:"varchar(100)" json:"createdTime"`
	UpdatedTime string `xorm:"varchar(100)" json:"updatedTime"`
	DisplayName string `xorm:"varchar(100)" json:"displayName"`

	Url   string      `xorm:"varchar(500)" json:"url"`
	Token string      `xorm:"varchar(500)" json:"-"`
	Tools []*McpTool  `xorm:"mediumtext" json:"tools"`

	ConfigText  string            `xorm:"mediumtext" json:"configText"`
	McpTools    []*agent.McpTools `xorm:"text" json:"mcpTools"`
	TestContent string            `xorm:"varchar(500)" json:"testContent"`
	IsDefault   bool              `json:"isDefault"`
}

func (s *Server) GetId() string {
	return fmt.Sprintf("%s/%s", s.Owner, s.Name)
}

func GetServers(owner string) ([]*Server, error) {
	servers := []*Server{}
	err := adapter.engine.Desc("created_time").Find(&servers, &Server{Owner: owner})
	if err != nil {
		return servers, err
	}
	return servers, nil
}

func getServer(owner, name string) (*Server, error) {
	server := Server{Owner: owner, Name: name}
	existed, err := adapter.engine.Get(&server)
	if err != nil {
		return &server, err
	}
	if existed {
		return &server, nil
	}
	return nil, nil
}

func GetServer(id string) (*Server, error) {
	owner, name, err := util.GetOwnerAndNameFromIdWithError(id)
	if err != nil {
		return nil, err
	}
	return getServer(owner, name)
}

func GetServerByOwnerAndName(owner, nameOrId string) (*Server, error) {
	if nameOrId == "" {
		return nil, nil
	}
	var id string
	if _, _, err := util.GetOwnerAndNameFromIdWithError(nameOrId); err == nil {
		id = nameOrId
	} else {
		id = util.GetIdFromOwnerAndName(owner, nameOrId)
	}
	s, err := GetServer(id)
	if err != nil {
		return nil, err
	}
	if s != nil {
		return s, nil
	}
	if owner != "admin" && !strings.Contains(nameOrId, "/") {
		return GetServer(util.GetIdFromOwnerAndName("admin", nameOrId))
	}
	return nil, nil
}

func AddServer(server *Server) (bool, error) {
	affected, err := adapter.engine.Insert(server)
	if err != nil {
		return false, err
	}
	return affected != 0, nil
}

func UpdateServer(id string, server *Server) (bool, error) {
	owner, name, err := util.GetOwnerAndNameFromIdWithError(id)
	if err != nil {
		return false, err
	}

	oldServer, err := getServer(owner, name)
	if err != nil {
		return false, err
	}
	if oldServer != nil && server.Token == "" {
		server.Token = oldServer.Token
	}

	_, err = adapter.engine.ID(core.PK{owner, name}).AllCols().Update(server)
	if err != nil {
		return false, err
	}
	return true, nil
}

func SyncMcpTool(id string, server *Server, isCleared bool) (bool, error) {
	owner, name, err := util.GetOwnerAndNameFromIdWithError(id)
	if err != nil {
		return false, err
	}

	if isCleared {
		server.Tools = nil
		_, err = adapter.engine.ID(core.PK{owner, name}).Cols("tools").Update(server)
		if err != nil {
			return false, err
		}
		return true, nil
	}

	oldServer, err := getServer(owner, name)
	if err != nil {
		return false, err
	}
	if oldServer == nil {
		return false, nil
	}
	if server.Token == "" {
		server.Token = oldServer.Token
	}

	if err = syncServerTools(server); err != nil {
		return false, err
	}

	_, err = adapter.engine.ID(core.PK{owner, name}).AllCols().Update(server)
	if err != nil {
		return false, err
	}
	return true, nil
}

func syncServerTools(server *Server) error {
	if server.Url == "" {
		return fmt.Errorf("server URL is empty")
	}

	oldTools := server.Tools
	if oldTools == nil {
		oldTools = []*McpTool{}
	}

	tools, err := mcppkg.GetToolsFromURL(server.Url, server.Token)
	if err != nil {
		return err
	}

	newTools := make([]*McpTool, 0, len(tools))
	for _, t := range tools {
		isAllowed := true
		for _, old := range oldTools {
			if old.Name == t.Name {
				isAllowed = old.IsAllowed
				break
			}
		}
		newTools = append(newTools, &McpTool{
			Name:        t.Name,
			Description: t.Description,
			IsAllowed:   isAllowed,
		})
	}

	server.Tools = newTools
	return nil
}

func DeleteServer(server *Server) (bool, error) {
	affected, err := adapter.engine.ID(core.PK{server.Owner, server.Name}).Delete(&Server{})
	if err != nil {
		return false, err
	}
	return affected != 0, nil
}

func (s *Server) GetAgentClients() (*agent.AgentClients, error) {
	toolsMap := make(map[string]bool)
	for _, tool := range s.McpTools {
		toolsMap[tool.ServerName] = tool.IsEnabled
	}
	clients, err := agent.GetMCPClientMap(s.ConfigText, toolsMap)
	if err != nil {
		return nil, err
	}
	var tools []*protocol.Tool
	for _, mcpTool := range s.McpTools {
		if !mcpTool.IsEnabled {
			continue
		}
		var toolsList []*protocol.Tool
		if err := json.Unmarshal([]byte(mcpTool.Tools), &toolsList); err != nil {
			return nil, err
		}
		for _, tool := range toolsList {
			tool.Name = agent.GetIdFromServerNameAndToolName(mcpTool.ServerName, tool.Name)
		}
		tools = append(tools, toolsList...)
	}
	return &agent.AgentClients{
		Clients: clients,
		Tools:   tools,
	}, nil
}

func RefreshServerMcpTools(server *Server) error {
	tools, err := agent.GetToolsList(server.ConfigText)
	if err != nil {
		return err
	}
	server.McpTools = tools
	return nil
}

func TestMcpServer(s *Server, lang string) (string, error) {
	if strings.TrimSpace(s.ConfigText) == "" {
		return "", fmt.Errorf("MCP server configuration (configText) is empty")
	}
	var payload struct {
		Tool      string                 `json:"tool"`
		Arguments map[string]interface{} `json:"arguments"`
	}
	if err := json.Unmarshal([]byte(s.TestContent), &payload); err != nil {
		return "", fmt.Errorf(i18n.Translate(lang, "object:invalid MCP test JSON in testContent: %v"), err)
	}
	if strings.TrimSpace(payload.Tool) == "" {
		return "", fmt.Errorf(i18n.Translate(lang, "object:MCP test JSON must include non-empty \"tool\""))
	}
	if payload.Arguments == nil {
		payload.Arguments = map[string]interface{}{}
	}
	return agent.TestMcpToolCall(s.ConfigText, s.McpTools, payload.Tool, payload.Arguments)
}

func GetMcpAgentClientsFromContext(owner, serverName, lang string) (*agent.AgentClients, error) {
	if serverName == "" {
		return nil, nil
	}
	server, err := GetServerByOwnerAndName(owner, serverName)
	if err != nil {
		return nil, err
	}
	if server == nil {
		return nil, fmt.Errorf(i18n.Translate(lang, "object:The MCP server: %s is not found"), serverName)
	}
	return server.GetAgentClients()
}

func GetServerCount(owner, field, value string) (int64, error) {
	session := GetDbSession(owner, -1, -1, field, value, "", "")
	count, err := session.Count(&Server{})
	if err != nil {
		return 0, err
	}
	return count, nil
}

func GetPaginationServers(owner string, offset, limit int, field, value, sortField, sortOrder string) ([]*Server, error) {
	servers := []*Server{}
	session := GetDbSession(owner, offset, limit, field, value, sortField, sortOrder)
	err := session.Find(&servers)
	if err != nil {
		return servers, err
	}
	return servers, nil
}
