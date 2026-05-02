// Copyright 2026 The OpenAgent Authors. All Rights Reserved.
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

import React from "react";
import {Button, Col, Row} from "antd";
import * as Setting from "../Setting";
import i18next from "i18next";
import * as ServerBackend from "../backend/ServerBackend";
import Editor from "./Editor";

function buildDefaultMcpTestJson(server) {
  if (server.mcpTools && server.mcpTools.length > 0) {
    const mt = server.mcpTools.find(t => t.isEnabled !== false) || server.mcpTools[0];
    try {
      const tools = JSON.parse(mt.tools || "[]");
      if (tools.length > 0 && tools[0].name) {
        const toolId = `${mt.serverName}__${tools[0].name}`;
        return JSON.stringify({tool: toolId, arguments: {}}, null, 2);
      }
    } catch (e) {
      // ignore
    }
  }
  return "{\n  \"tool\": \"serverName__toolName\",\n  \"arguments\": {}\n}";
}

class TestMcpWidget extends React.Component {
  constructor(props) {
    super(props);
    this.state = {
      testButtonLoading: false,
      testResult: "",
    };
  }

  componentDidMount() {
    this.syncFromServer(this.props.server, null);
  }

  componentDidUpdate(prevProps) {
    const {server} = this.props;
    if (server !== prevProps.server) {
      this.syncFromServer(server, prevProps.server);
    }
  }

  syncFromServer(server, prevServer) {
    const {onUpdateServer} = this.props;
    if (!server) {
      return;
    }
    if (!server.testContent || server.testContent.trim() === "") {
      const def = buildDefaultMcpTestJson(server);
      if (onUpdateServer) {
        onUpdateServer("testContent", def);
      }
    }
  }

  async sendTestMcp(server) {
    let parsed;
    try {
      parsed = JSON.parse(server.testContent);
    } catch (e) {
      Setting.showMessage("error", `${i18next.t("provider:Invalid MCP test JSON")}: ${e.message}`);
      return;
    }
    if (!parsed || typeof parsed.tool !== "string" || parsed.tool.trim() === "") {
      Setting.showMessage("error", i18next.t("provider:MCP test JSON must include tool"));
      return;
    }

    this.setState({testButtonLoading: true, testResult: ""});

    try {
      const res = await ServerBackend.testMcpServer(server);
      if (res.status === "ok") {
        const out = typeof res.data === "string" ? res.data : JSON.stringify(res.data, null, 2);
        this.setState({testResult: out});
        Setting.showMessage("success", i18next.t("general:Success"));
      } else {
        Setting.showMessage("error", res.msg || i18next.t("general:Failed to save"));
      }
    } catch (error) {
      Setting.showMessage("error", `${i18next.t("general:Failed to connect to server")}: ${error.message}`);
    } finally {
      this.setState({testButtonLoading: false});
    }
  }

  render() {
    const {server, onUpdateServer} = this.props;

    if (!server) {
      return null;
    }

    return (
      <React.Fragment>
        <Row style={{marginTop: "20px"}} >
          <Col style={{marginTop: "5px"}} span={(Setting.isMobile()) ? 22 : 2}>
            {Setting.getLabel(i18next.t("provider:Provider test"), i18next.t("provider:MCP test JSON - Tooltip"))} :
          </Col>
          <Col span={10} >
            <Editor
              value={server.testContent}
              lang="json"
              height="150px"
              dark
              onChange={value => {onUpdateServer("testContent", value);}}
            />
          </Col>
          <Col span={6} >
            <Button
              style={{marginLeft: "10px", marginBottom: "5px"}}
              type="primary"
              loading={this.state.testButtonLoading}
              disabled={!server.testContent || server.testContent.trim() === ""}
              onClick={() => this.sendTestMcp(server)}
            >
              {i18next.t("provider:Invoke MCP tool")}
            </Button>
          </Col>
        </Row>
        {this.state.testResult ? (
          <Row style={{marginTop: "10px"}}>
            <Col span={2}></Col>
            <Col span={10}>
              <div style={{marginBottom: "5px"}}><strong>{i18next.t("provider:MCP tool result")}:</strong></div>
              <Editor
                value={this.state.testResult}
                lang="text"
                height="150px"
                dark
                readOnly
              />
            </Col>
          </Row>
        ) : null}
      </React.Fragment>
    );
  }
}

export default TestMcpWidget;
