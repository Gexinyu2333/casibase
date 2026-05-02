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

import React from "react";
import Loading from "./common/Loading";
import {Button, Card, Col, Input, Row} from "antd";
import {LinkOutlined} from "@ant-design/icons";
import * as ServerBackend from "./backend/ServerBackend";
import * as Setting from "./Setting";
import i18next from "i18next";
import ToolTable from "./table/ToolTable";
import TestMcpWidget from "./common/TestMcpWidget";

class ServerEditPage extends React.Component {
  constructor(props) {
    super(props);
    this.state = {
      classes: props,
      serverName: props.match.params.serverName,
      server: null,
      originalServer: null,
      isNewServer: props.location?.state?.isNewServer || false,
      syncButtonLoading: false,
    };
  }

  UNSAFE_componentWillMount() {
    this.getServer();
  }

  getServer() {
    ServerBackend.getServer("admin", this.state.serverName)
      .then((res) => {
        if (res.status === "ok") {
          this.setState({
            server: res.data,
            originalServer: Setting.deepCopy(res.data),
          });
        } else {
          Setting.showMessage("error", `${i18next.t("general:Failed to get")}: ${res.msg}`);
        }
      });
  }

  updateServerField(key, value) {
    const server = this.state.server;
    server[key] = value;
    this.setState({server});
  }

  submitServerEdit(willExist) {
    const server = Setting.deepCopy(this.state.server);
    ServerBackend.updateServer(this.state.originalServer.owner, this.state.originalServer.name, server)
      .then((res) => {
        if (res.status === "ok") {
          Setting.showMessage("success", i18next.t("general:Successfully saved"));
          this.setState({originalServer: Setting.deepCopy(this.state.server)});
          if (willExist) {
            this.props.history.push("/servers");
          }
        } else {
          Setting.showMessage("error", `${i18next.t("general:Failed to save")}: ${res.msg}`);
        }
      })
      .catch(error => {
        Setting.showMessage("error", `${i18next.t("general:Failed to connect to server")}: ${error}`);
      });
  }

  cancelServerEdit() {
    ServerBackend.deleteServer(this.state.server)
      .then(() => {
        this.props.history.push("/servers");
      });
  }

  syncMcpTool(isCleared) {
    const server = Setting.deepCopy(this.state.server);
    this.setState({syncButtonLoading: true});
    ServerBackend.syncMcpTool(this.state.originalServer.owner, this.state.originalServer.name, server, isCleared)
      .then((res) => {
        if (res.status === "ok") {
          Setting.showMessage("success", i18next.t("general:Successfully saved"));
          this.getServer();
        } else {
          Setting.showMessage("error", `${i18next.t("general:Failed to save")}: ${res.msg}`);
        }
      })
      .catch(error => {
        Setting.showMessage("error", `${i18next.t("general:Failed to connect to server")}: ${error}`);
      })
      .finally(() => {
        this.setState({syncButtonLoading: false});
      });
  }

  renderServer() {
    return (
      <Card size="small" title={
        <div>
          {i18next.t("server:Edit MCP Server")}&nbsp;&nbsp;&nbsp;&nbsp;
          <Button onClick={() => this.submitServerEdit(false)}>{i18next.t("general:Save")}</Button>
          <Button style={{marginLeft: "20px"}} type="primary" onClick={() => this.submitServerEdit(true)}>{i18next.t("general:Save & Exit")}</Button>
          {this.state.isNewServer && <Button style={{marginLeft: "20px"}} onClick={() => this.cancelServerEdit()}>{i18next.t("general:Cancel")}</Button>}
        </div>
      } style={{marginLeft: "5px"}} type="inner">
        <Row style={{marginTop: "10px"}}>
          <Col style={{marginTop: "5px"}} span={(Setting.isMobile()) ? 22 : 2}>
            {Setting.getLabel(i18next.t("general:Name"), i18next.t("general:Name - Tooltip"))} :
          </Col>
          <Col span={22}>
            <Input value={this.state.server.name} onChange={e => this.updateServerField("name", e.target.value)} />
          </Col>
        </Row>
        <Row style={{marginTop: "20px"}}>
          <Col style={{marginTop: "5px"}} span={(Setting.isMobile()) ? 22 : 2}>
            {Setting.getLabel(i18next.t("general:Display name"), i18next.t("general:Display name - Tooltip"))} :
          </Col>
          <Col span={22}>
            <Input value={this.state.server.displayName} onChange={e => this.updateServerField("displayName", e.target.value)} />
          </Col>
        </Row>
        <Row style={{marginTop: "20px"}}>
          <Col style={{marginTop: "5px"}} span={(Setting.isMobile()) ? 22 : 2}>
            {Setting.getLabel(i18next.t("general:URL"), i18next.t("general:URL - Tooltip"))} :
          </Col>
          <Col span={22}>
            <Input prefix={<LinkOutlined />} value={this.state.server.url} onChange={e => this.updateServerField("url", e.target.value)} />
          </Col>
        </Row>
        <Row style={{marginTop: "20px"}}>
          <Col style={{marginTop: "5px"}} span={(Setting.isMobile()) ? 22 : 2}>
            {Setting.getLabel(i18next.t("server:Access token"), i18next.t("server:Access token - Tooltip"))} :
          </Col>
          <Col span={22}>
            <Input.Password placeholder={"***"} value={this.state.server.token} onChange={e => this.updateServerField("token", e.target.value)} />
          </Col>
        </Row>
        <Row style={{marginTop: "20px"}}>
          <Col style={{marginTop: "5px"}} span={(Setting.isMobile()) ? 22 : 2}>
            {Setting.getLabel(i18next.t("general:Tool"), i18next.t("general:Tool - Tooltip"))} :
          </Col>
          <Col span={22}>
            {!this.state.isNewServer && (
              <>
                <Button type="primary" loading={this.state.syncButtonLoading} style={{marginBottom: "5px"}} onClick={() => this.syncMcpTool(false)}>{i18next.t("general:Sync")}</Button>
                <Button style={{marginBottom: "5px", marginLeft: "10px"}} onClick={() => this.syncMcpTool(true)}>{i18next.t("general:Clear")}</Button>
              </>
            )}
            <ToolTable
              tools={this.state.server.tools || []}
              onUpdateTable={(value) => this.updateServerField("tools", value)}
            />
          </Col>
        </Row>
        <Row style={{marginTop: "20px"}}>
          <Col style={{marginTop: "5px"}} span={(Setting.isMobile()) ? 22 : 2}>
            {Setting.getLabel(i18next.t("server:MCP test"), i18next.t("server:MCP test - Tooltip"))} :
          </Col>
          <Col span={22}>
            <TestMcpWidget server={this.state.server} />
          </Col>
        </Row>
        <Row style={{marginTop: "20px"}}>
          <Col style={{marginTop: "5px"}} span={(Setting.isMobile()) ? 22 : 2}>
            {Setting.getLabel(i18next.t("server:Base URL"), i18next.t("server:Base URL - Tooltip"))} :
          </Col>
          <Col span={22}>
            <Input prefix={<LinkOutlined />} readOnly value={`${window.location.origin}/api/get-server?id=${this.state.server.owner}/${this.state.server.name}`} />
          </Col>
        </Row>
      </Card>
    );
  }

  render() {
    return (
      <div>
        {this.state.server === null ? <Loading /> : this.renderServer()}
      </div>
    );
  }
}

export default ServerEditPage;
