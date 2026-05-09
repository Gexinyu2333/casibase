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
import {Button, Card, Col, DatePicker, Input, Row, Select, Space} from "antd";
import * as GraphBackend from "./backend/GraphBackend";
import * as ChatBackend from "./backend/ChatBackend";
import * as StoreBackend from "./backend/StoreBackend";
import * as Setting from "./Setting";
import i18next from "i18next";
import GraphDataPage from "./GraphDataPage";
import GraphChatDataPage from "./GraphChatDataPage";
import GraphChatTable from "./GraphChatTable";
import Editor from "./common/Editor";
import dayjs from "dayjs";
import weekday from "dayjs/plugin/weekday";
import localeData from "dayjs/plugin/localeData";

dayjs.extend(weekday);
dayjs.extend(localeData);

class GraphEditPage extends React.Component {
  constructor(props) {
    super(props);
    this.state = {
      classes: props,
      graphName: props.match.params.graphName,
      isNewGraph: props.location?.state?.isNewGraph || false,
      graph: null,
      graphCount: "key",
      stores: [],
      filteredChats: [],
      tempStartTime: null,
      tempEndTime: null,
    };
  }

  UNSAFE_componentWillMount() {
    this.getGraph();
    this.getStores();
  }

  getStores() {
    StoreBackend.getStores(this.props.account.name)
      .then((res) => {
        if (res.status === "ok") {
          this.setState({
            stores: res.data || [],
          });
        }
      });
  }

  getGraph() {
    GraphBackend.getGraph(this.props.account.name, this.state.graphName)
      .then((res) => {
        if (res.status === "ok") {
          this.setState({
            graph: res.data,
          }, () => {
            this.loadFilteredChats();
          });
        } else {
          Setting.showMessage("error", `${i18next.t("general:Failed to get")}: ${res.msg}`);
        }
      });
  }

  parseGraphField(key, value) {
    if (["density"].includes(key)) {
      value = parseFloat(value);
    }
    return value;
  }

  updateGraphField(key, value) {
    value = this.parseGraphField(key, value);

    const graph = this.state.graph;
    graph[key] = value;
    this.setState({
      graph: graph,
    });
  }

  handleErrorChange(errorText) {
    this.updateGraphField("errorText", errorText);
  }

  generateGraphData() {
    this.updateGraphField("text", "");

    const graph = Setting.deepCopy(this.state.graph);
    graph.text = "";

    GraphBackend.updateGraph(this.state.graph.owner, this.state.graphName, graph)
      .then((res) => {
        if (res.status === "ok") {
          this.getGraph();
          this.loadFilteredChats();
          Setting.showMessage("success", i18next.t("general:Successfully generated"));
        } else {
          Setting.showMessage("error", `${i18next.t("general:Failed to generate")}: ${res.msg}`);
        }
      })
      .catch(error => {
        Setting.showMessage("error", `${i18next.t("general:Failed to generate")}: ${error}`);
      });
  }

  loadFilteredChats() {
    if (this.state.graph && this.state.graph.category === "Chats") {
      ChatBackend.getChats("admin", this.state.graph.store, "", "", "", "", "", "", "", this.state.graph.startTime, this.state.graph.endTime)
        .then((res) => {
          if (res.status === "ok") {
            this.setState({
              filteredChats: res.data || [],
            });
          }
        });
    }
  }

  toDayjs = (rfc3339) => {
    if (!rfc3339) {return null;}
    const d = dayjs(rfc3339);
    return d.isValid() ? d : null;
  };

  toRFC3339 = (d) => (d ? d.format("YYYY-MM-DDTHH:mm:ssZ") : "");

  renderGraphField(label, control, span = 8) {
    return (
      <Col style={{marginTop: "12px"}} span={Setting.isMobile() ? 22 : span}>
        <div style={{marginBottom: "6px", color: "#595959", fontWeight: 500, lineHeight: "22px", fontSize: "13px"}}>{label}</div>
        {control}
      </Col>
    );
  }

  renderGraphActions() {
    const btnStyle = {
      backgroundColor: "#F8F9FA",
      borderColor: "rgb(229, 229, 234)",
      border: "1px solid #E5E5EA",
      borderRadius: "10px",
      padding: "6px 10px",
    };
    return (
      <Space wrap>
        <Button style={btnStyle} onClick={() => this.submitGraphEdit(false)}>{i18next.t("general:Save")}</Button>
        <Button style={btnStyle} onClick={() => this.submitGraphEdit(true)}>{i18next.t("general:Save & Exit")}</Button>
        {this.state.isNewGraph && <Button style={btnStyle} onClick={() => this.cancelGraphEdit()}>{i18next.t("general:Cancel")}</Button>}
      </Space>
    );
  }

  renderGraph() {
    const graph = this.state.graph;
    const rowGutter = [16, 8];
    const cardHeadStyle = {background: "transparent", borderBottom: "none", fontWeight: 600, fontSize: "15px", fontFamily: "Inter, -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif"};
    const sectionCardStyle = {
      marginBottom: "16px",
      borderRadius: "14px",
      boxShadow: "0 1px 3px rgba(0,0,0,0.06), 0 1px 2px rgba(0,0,0,0.04)",
      padding: "18px",
    };
    const renderCardTitle = (title, desc) => (
      <div>
        <div style={{fontWeight: 600, fontSize: "15px"}}>{title}</div>
        {desc && <div style={{fontSize: "13px", color: "#6E6E73", fontWeight: 400, marginTop: "2px"}}>{desc}</div>}
      </div>
    );

    return (
      <div>
        <div style={{marginBottom: "16px", display: "flex", justifyContent: "space-between", alignItems: "center"}}>
          <span style={{fontSize: "22px", fontWeight: 600}}>{i18next.t("graph:Edit Graph")}</span>
          <div style={{display: "flex", gap: "8px", marginRight: "4px"}}>
            {this.renderGraphActions()}
          </div>
        </div>

        <Card size="small" title={renderCardTitle(i18next.t("general:General Settings"), i18next.t("general:General Settings desc"))} style={sectionCardStyle} headStyle={cardHeadStyle}>
          <Row gutter={rowGutter}>
            {this.renderGraphField(
              Setting.getLabel(i18next.t("general:Name"), i18next.t("general:Name - Tooltip")),
              <Input value={graph.name} onChange={(e) => this.updateGraphField("name", e.target.value)} />,
              8
            )}
            {this.renderGraphField(
              Setting.getLabel(i18next.t("general:Display name"), i18next.t("general:Display name - Tooltip")),
              <Input value={graph.displayName} onChange={(e) => this.updateGraphField("displayName", e.target.value)} />,
              8
            )}
            {this.renderGraphField(
              Setting.getLabel(i18next.t("general:Category"), i18next.t("provider:Category - Tooltip")),
              <Select
                style={{width: "100%"}}
                value={graph.category || "Default"}
                onChange={(value) => this.updateGraphField("category", value)}
              >
                <Select.Option value="Default">{i18next.t("general:Default")}</Select.Option>
                <Select.Option value={"Assets"}>{i18next.t("general:Assets")}</Select.Option>
                <Select.Option value={"Chats"}>{i18next.t("general:Chats")}</Select.Option>
              </Select>,
              8
            )}
            {this.renderGraphField(
              Setting.getLabel(i18next.t("graph:Layout"), i18next.t("graph:Layout - Tooltip")),
              <Select
                style={{width: "100%"}}
                value={graph.layout || (graph.category === "Chats" ? "wordcloud" : "force")}
                onChange={(value) => this.updateGraphField("layout", value)}
              >
                {graph.category === "Chats" ? (
                  <Select.Option value="wordcloud">{i18next.t("graph:Word Cloud")}</Select.Option>
                ) : (
                  <>
                    <Select.Option value="force">{i18next.t("graph:Force")}</Select.Option>
                    <Select.Option value="circular">{i18next.t("graph:Circular")}</Select.Option>
                    <Select.Option value="radial">{i18next.t("graph:Radial")}</Select.Option>
                    <Select.Option value="grid">{i18next.t("graph:Grid")}</Select.Option>
                    <Select.Option value="tree">{i18next.t("graph:Tree")}</Select.Option>
                    <Select.Option value="none">{i18next.t("general:None")}</Select.Option>
                  </>
                )}
              </Select>,
              8
            )}
            {graph.category === "Chats" && (
              <>
                {this.renderGraphField(
                  Setting.getLabel(i18next.t("general:Store"), i18next.t("general:Store - Tooltip")),
                  <Select
                    style={{width: "100%"}}
                    value={graph.store}
                    onChange={(value) => this.updateGraphField("store", value)}
                    allowClear
                  >
                    {this.state.stores.map((store) => (
                      <Select.Option key={store.name} value={store.name}>{store.displayName || store.name}</Select.Option>
                    ))}
                  </Select>,
                  8
                )}
                {this.renderGraphField(
                  Setting.getLabel(i18next.t("video:Start time (s)"), i18next.t("video:Start time (s) - Tooltip")),
                  <DatePicker
                    showTime
                    style={{width: "100%"}}
                    value={this.toDayjs(graph.startTime)}
                    onChange={(date) => this.updateGraphField("startTime", this.toRFC3339(date))}
                  />,
                  8
                )}
                {this.renderGraphField(
                  Setting.getLabel(i18next.t("video:End time (s)"), i18next.t("video:End time (s) - Tooltip")),
                  <DatePicker
                    showTime
                    style={{width: "100%"}}
                    value={this.toDayjs(graph.endTime)}
                    onChange={(date) => this.updateGraphField("endTime", this.toRFC3339(date))}
                  />,
                  8
                )}
                {this.renderGraphField(
                  Setting.getLabel(i18next.t("graph:Threshold"), i18next.t("graph:Threshold - Tooltip")),
                  <Input
                    type="number"
                    min={1}
                    step={1}
                    value={graph.density || 1}
                    onChange={(e) => this.updateGraphField("density", e.target.value)}
                  />,
                  8
                )}
                <Col style={{marginTop: "12px"}} span={24}>
                  <Button type="primary" onClick={() => this.generateGraphData()}>
                    {i18next.t("general:Generate")}
                  </Button>
                </Col>
              </>
            )}
            {graph.category !== "Chats" && this.renderGraphField(
              Setting.getLabel(i18next.t("graph:Node density"), i18next.t("graph:Node density - Tooltip")),
              <Input
                type="number"
                min={0.1}
                max={10}
                step={0.1}
                value={graph.density || 5}
                onChange={(e) => this.updateGraphField("density", e.target.value)}
              />,
              8
            )}
          </Row>
        </Card>

        <Card size="small" title={renderCardTitle(i18next.t("general:Content"), "")} style={sectionCardStyle} headStyle={cardHeadStyle}>
          <Row gutter={rowGutter}>
            {this.renderGraphField(
              Setting.getLabel(i18next.t("general:Text"), i18next.t("general:Text - Tooltip")),
              <div style={{height: "500px"}}>
                <Editor
                  value={graph.text}
                  lang="json"
                  fillHeight
                  dark
                  onChange={(value) => this.updateGraphField("text", value)}
                />
              </div>,
              24
            )}
          </Row>
        </Card>

        <Card size="small" title={renderCardTitle(i18next.t("general:Preview"), "")} style={sectionCardStyle} headStyle={cardHeadStyle}>
          <div key={this.state.graphCount} style={{height: "1000px", width: "100%"}}>
            {graph.category === "Chats" ? (
              <GraphChatDataPage graphText={graph.text} showBorder={true} onErrorChange={(errorText) => this.handleErrorChange(errorText)} />
            ) : (
              <GraphDataPage account={this.props.account} owner={graph.owner} graphName={graph.name} graphText={graph.text} category={graph.category} layout={graph.layout} density={graph.density} showBorder={true} onErrorChange={(errorText) => this.handleErrorChange(errorText)} />
            )}
          </div>
        </Card>
      </div>
    );
  }

  renderFilteredChatsSection() {
    if (!this.state.graph || this.state.graph.category !== "Chats") {
      return null;
    }

    return <GraphChatTable chats={this.state.filteredChats} />;
  }

  submitGraphEdit(exitAfterSave) {
    const graph = Setting.deepCopy(this.state.graph);
    if (!exitAfterSave) {
      this.setState({
        graphCount: this.state.graphCount + "a",
      });
    }
    GraphBackend.updateGraph(this.state.graph.owner, this.state.graphName, graph)
      .then((res) => {
        if (res.status === "ok") {
          if (res.data) {
            Setting.showMessage("success", i18next.t("general:Successfully saved"));
            this.setState({
              graphName: this.state.graph.name,
              isNewGraph: false,
            });
            if (exitAfterSave) {
              this.props.history.push("/graphs");
            } else {
              this.props.history.push(`/graphs/${this.state.graph.name}`);
            }
          } else {
            Setting.showMessage("error", i18next.t("general:Failed to save"));
            this.updateGraphField("name", this.state.graphName);
          }
        } else {
          Setting.showMessage("error", `${i18next.t("general:Failed to save")}: ${res.msg}`);
        }
      })
      .catch(error => {
        Setting.showMessage("error", `${i18next.t("general:Failed to save")}: ${error}`);
      });
  }

  cancelGraphEdit() {
    if (this.state.isNewGraph) {
      GraphBackend.deleteGraph(this.state.graph)
        .then((res) => {
          if (res.status === "ok") {
            Setting.showMessage("success", i18next.t("general:Cancelled successfully"));
            this.props.history.push("/graphs");
          } else {
            Setting.showMessage("error", `${i18next.t("general:Failed to cancel")}: ${res.msg}`);
          }
        })
        .catch(error => {
          Setting.showMessage("error", `${i18next.t("general:Failed to cancel")}: ${error}`);
        });
    } else {
      this.props.history.push("/graphs");
    }
  }

  render() {
    return (
      <div style={{background: "#F1F3F5", padding: "16px 20px 32px", minHeight: "100vh"}}>
        {this.state.graph !== null ? this.renderGraph() : <Loading type="page" tip={i18next.t("general:Loading")} />}
        {this.renderFilteredChatsSection()}
      </div>
    );
  }
}

export default GraphEditPage;
