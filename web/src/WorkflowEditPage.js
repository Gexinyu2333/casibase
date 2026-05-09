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
import {Button, Card, Col, Input, Mentions, Popover, Row, Space} from "antd";
import * as WorkflowBackend from "./backend/WorkflowBackend";
import * as Setting from "./Setting";
import i18next from "i18next";
import BpmnComponent from "./BpmnComponent";
import Editor from "./common/Editor";

import ChatWidget from "./common/ChatWidget";

class WorkflowEditPage extends React.Component {
  constructor(props) {
    super(props);
    this.state = {
      classes: props,
      workflowName: props.match.params.workflowName,
      isNewWorkflow: props.location?.state?.isNewWorkflow || false,
      modelProviders: [],
      workflow: null,
      chatPageObj: null,
      loading: false,
    };

    this.questionTemplatesOptions = [
      {value: "{{text}}", label: i18next.t("general:Text")},
      {value: "{{text2}}", label: i18next.t("general:Text2")},
      {value: "{{message}}", label: i18next.t("general:Message")},
      {value: "{{language}}", label: i18next.t("general:Language")},
    ];
  }

  UNSAFE_componentWillMount() {
    this.getWorkflow();
  }

  getWorkflow() {
    WorkflowBackend.getWorkflow(this.props.account.name, this.state.workflowName)
      .then((res) => {
        if (res.status === "ok") {
          this.setState({
            workflow: res.data,
          });
        } else {
          Setting.showMessage("error", `${i18next.t("general:Failed to get")}: ${res.msg}`);
        }
      });
  }

  parseWorkflowField(key, value) {
    if ([""].includes(key)) {
      value = Setting.myParseInt(value);
    }
    return value;
  }

  updateWorkflowField(key, value) {
    value = this.parseWorkflowField(key, value);

    const workflow = this.state.workflow;
    workflow[key] = value;
    this.setState({
      workflow: workflow,
    });
  }

  renderQuestionTemplate() {
    const questionTemplate = this.state.workflow.questionTemplate;

    if (!questionTemplate) {
      return "";
    }

    const renderedTemplate = questionTemplate.replace(/#\{\{(\w+)\}\}/g, (match, variableName) => {
      if (variableName === "language") {
        const lang = Setting.getLanguage();
        return (!lang || lang === "null") ? "en" : lang;
      }
      return this.state.workflow[variableName] || "";
    });

    return renderedTemplate;
  }

  renderWorkflowField(label, control, span = 8) {
    return (
      <Col style={{marginTop: "12px"}} span={Setting.isMobile() ? 22 : span}>
        <div style={{marginBottom: "6px", color: "var(--ant-color-text-secondary)", fontWeight: 500, lineHeight: "22px", fontSize: "13px"}}>{label}</div>
        {control}
      </Col>
    );
  }

  renderWorkflowActions() {
    const btnStyle = {
      backgroundColor: "var(--ant-color-bg-container)",
      borderColor: "var(--ant-color-border)",
      border: "1px solid var(--ant-color-border)",
      borderRadius: "10px",
      padding: "6px 10px",
    };
    return (
      <Space wrap>
        <Button style={btnStyle} onClick={() => this.submitWorkflowEdit(false)}>{i18next.t("general:Save")}</Button>
        <Button style={btnStyle} onClick={() => this.submitWorkflowEdit(true)}>{i18next.t("general:Save & Exit")}</Button>
        {this.state.isNewWorkflow && <Button style={btnStyle} onClick={() => this.cancelWorkflowEdit()}>{i18next.t("general:Cancel")}</Button>}
      </Space>
    );
  }

  renderWorkflow() {
    const workflow = this.state.workflow;
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
        {desc && <div style={{fontSize: "13px", color: "var(--ant-color-text-tertiary)", fontWeight: 400, marginTop: "2px"}}>{desc}</div>}
      </div>
    );

    return (
      <div>
        <div style={{marginBottom: "16px", display: "flex", justifyContent: "space-between", alignItems: "center"}}>
          <span style={{fontSize: "22px", fontWeight: 600}}>{i18next.t("workflow:Edit Workflow")}</span>
          <div style={{display: "flex", gap: "8px", marginRight: "4px"}}>
            {this.renderWorkflowActions()}
          </div>
        </div>

        <Card size="small" title={renderCardTitle(i18next.t("general:General Settings"), i18next.t("general:General Settings desc"))} style={sectionCardStyle} headStyle={cardHeadStyle}>
          <Row gutter={rowGutter}>
            {this.renderWorkflowField(
              Setting.getLabel(i18next.t("general:Name"), i18next.t("general:Name - Tooltip")),
              <Input value={workflow.name} onChange={(e) => this.updateWorkflowField("name", e.target.value)} />,
              8
            )}
            {this.renderWorkflowField(
              Setting.getLabel(i18next.t("general:Display name"), i18next.t("general:Display name - Tooltip")),
              <Input value={workflow.displayName} onChange={(e) => this.updateWorkflowField("displayName", e.target.value)} />,
              8
            )}
          </Row>
        </Card>

        <Card size="small" title={renderCardTitle(i18next.t("general:Content"), "")} style={sectionCardStyle} headStyle={cardHeadStyle}>
          <Row gutter={rowGutter}>
            {this.renderWorkflowField(
              Setting.getLabel(i18next.t("general:Text"), i18next.t("general:Text - Tooltip")),
              <Row gutter={8}>
                <Col span={10}>
                  <div style={{height: "500px"}}>
                    <Editor
                      value={workflow.text}
                      lang="xml"
                      fillHeight
                      fillWidth
                      dark
                      onChange={(value) => this.updateWorkflowField("text", value)}
                    />
                  </div>
                </Col>
                <Col span={1} />
                <Col span={13}>
                  <BpmnComponent
                    diagramXML={workflow.text}
                    onLoading={(info) => Setting.showMessage("success", info)}
                    onError={(err) => Setting.showMessage("error", err)}
                    onXMLChange={(xml) => this.updateWorkflowField("text", xml)}
                  />
                </Col>
              </Row>,
              24
            )}
            {this.renderWorkflowField(
              Setting.getLabel(i18next.t("general:Text2"), i18next.t("general:Text2 - Tooltip")),
              <Row gutter={8}>
                <Col span={10}>
                  <div style={{height: "500px"}}>
                    <Editor
                      value={workflow.text2}
                      lang="xml"
                      fillHeight
                      fillWidth
                      dark
                      onChange={(value) => this.updateWorkflowField("text2", value)}
                    />
                  </div>
                </Col>
                <Col span={1} />
                <Col span={13}>
                  <BpmnComponent
                    diagramXML={workflow.text2}
                    onLoading={(info) => Setting.showMessage("success", info)}
                    onError={(err) => Setting.showMessage("error", err)}
                    onXMLChange={(xml) => this.updateWorkflowField("text2", xml)}
                  />
                </Col>
              </Row>,
              24
            )}
            {this.renderWorkflowField(
              Setting.getLabel(i18next.t("general:Message"), i18next.t("general:Message - Tooltip")),
              <div style={{height: "500px"}}>
                <Editor
                  value={workflow.message}
                  lang="html"
                  fillHeight
                  fillWidth
                  dark
                  onChange={(value) => this.updateWorkflowField("message", value)}
                />
              </div>,
              24
            )}
            {this.renderWorkflowField(
              Setting.getLabel(i18next.t("general:Template"), i18next.t("general:Template - Tooltip")),
              <Popover placement="top" trigger="click"
                content={
                  <Row gutter={[16, 8]} style={{width: "1000px"}}>
                    <Col span={12}>
                      <div style={{marginBottom: "8px"}}>{i18next.t("general:Template")}:</div>
                      <Mentions rows={25} prefix={"#"} options={this.questionTemplatesOptions} value={workflow.questionTemplate} onChange={(value) => this.updateWorkflowField("questionTemplate", value)} />
                    </Col>
                    <Col span={12}>
                      <div style={{marginBottom: "8px"}}>{i18next.t("general:Preview")}:</div>
                      <div style={{height: "600px", borderRadius: "4px"}}>
                        <Editor
                          value={this.renderQuestionTemplate()}
                          lang="markdown"
                          fillHeight
                          fillWidth
                          dark
                          readOnly
                        />
                      </div>
                    </Col>
                  </Row>
                }>
                <Input readOnly value={Setting.getShortText(workflow.questionTemplate, 60)} />
              </Popover>,
              24
            )}
            {this.renderWorkflowField(
              Setting.getLabel(i18next.t("general:Response"), i18next.t("general:Response - Tooltip")),
              <ChatWidget
                chatName={`workflow_chat_${this.state.workflowName}`}
                displayName={`${i18next.t("general:Chat")} - ${this.state.workflowName}`}
                category="Workflow"
                account={this.props.account}
                title={i18next.t("general:Chat")}
                height="800px"
                showNewChatButton={true}
                prompts={[{
                  "title": i18next.t("task:Generate Project"),
                  "text": this.renderQuestionTemplate(),
                  "image": "",
                }]}
              />,
              24
            )}
          </Row>
        </Card>
      </div>
    );
  }

  submitWorkflowEdit(exitAfterSave) {
    const workflow = Setting.deepCopy(this.state.workflow);
    WorkflowBackend.updateWorkflow(this.state.workflow.owner, this.state.workflowName, workflow)
      .then((res) => {
        if (res.status === "ok") {
          if (res.data) {
            Setting.showMessage("success", i18next.t("general:Successfully saved"));
            this.setState({
              workflowName: this.state.workflow.name,
              isNewWorkflow: false,
            });
            if (exitAfterSave) {
              this.props.history.push("/workflows");
            } else {
              this.props.history.push(`/workflows/${this.state.workflow.name}`);
              this.getWorkflow();
            }
          } else {
            Setting.showMessage("error", i18next.t("general:Failed to save"));
            this.updateWorkflowField("name", this.state.workflowName);
          }
        } else {
          Setting.showMessage("error", `${i18next.t("general:Failed to save")}: ${res.msg}`);
        }
      })
      .catch(error => {
        Setting.showMessage("error", `${i18next.t("general:Failed to save")}: ${error}`);
      });
  }

  cancelWorkflowEdit() {
    if (this.state.isNewWorkflow) {
      WorkflowBackend.deleteWorkflow(this.state.workflow)
        .then((res) => {
          if (res.status === "ok") {
            Setting.showMessage("success", i18next.t("general:Cancelled successfully"));
            this.props.history.push("/workflows");
          } else {
            Setting.showMessage("error", `${i18next.t("general:Failed to cancel")}: ${res.msg}`);
          }
        })
        .catch(error => {
          Setting.showMessage("error", `${i18next.t("general:Failed to cancel")}: ${error}`);
        });
    } else {
      this.props.history.push("/workflows");
    }
  }

  render() {
    return (
      <div style={{background: "var(--ant-color-bg-layout)", padding: "16px 20px 32px", minHeight: "100vh"}}>
        {this.state.workflow !== null ? this.renderWorkflow() : <Loading type="page" tip={i18next.t("general:Loading")} />}
      </div>
    );
  }
}

export default WorkflowEditPage;
