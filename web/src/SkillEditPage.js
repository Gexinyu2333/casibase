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
import Loading from "./common/Loading";
import {Button, Card, Col, Input, Row, Select} from "antd";
import * as SkillBackend from "./backend/SkillBackend";
import * as Setting from "./Setting";
import i18next from "i18next";

const {Option} = Select;
const {TextArea} = Input;

const SKILL_TYPES = ["writing", "coding", "analysis", "translation", "reasoning", "search", "custom"];

class SkillEditPage extends React.Component {
  constructor(props) {
    super(props);
    this.state = {
      classes: props,
      skillName: props.match.params.skillName,
      skill: null,
      originalSkill: null,
      isNewSkill: props.location?.state?.isNewSkill || false,
    };
  }

  UNSAFE_componentWillMount() {
    this.getSkill();
  }

  getSkill() {
    SkillBackend.getSkill("admin", this.state.skillName)
      .then((res) => {
        if (res.status === "ok") {
          this.setState({
            skill: res.data,
            originalSkill: Setting.deepCopy(res.data),
          });
        } else {
          Setting.showMessage("error", `${i18next.t("general:Failed to get")}: ${res.msg}`);
        }
      });
  }

  updateSkillField(key, value) {
    const skill = this.state.skill;
    skill[key] = value;
    this.setState({skill});
  }

  renderSkill() {
    return (
      <Card size="small" title={
        <div>
          {i18next.t("skill:Edit Skill")}&nbsp;&nbsp;&nbsp;&nbsp;
          <Button onClick={() => this.submitSkillEdit(false)}>{i18next.t("general:Save")}</Button>
          <Button style={{marginLeft: "20px"}} type="primary" onClick={() => this.submitSkillEdit(true)}>{i18next.t("general:Save & Exit")}</Button>
          {this.state.isNewSkill && <Button style={{marginLeft: "20px"}} onClick={() => this.cancelSkillEdit()}>{i18next.t("general:Cancel")}</Button>}
        </div>
      } style={{marginLeft: "5px"}} type="inner">
        <Row style={{marginTop: "10px"}}>
          <Col style={{marginTop: "5px"}} span={(Setting.isMobile()) ? 22 : 2}>
            {Setting.getLabel(i18next.t("general:Name"), i18next.t("general:Name - Tooltip"))} :
          </Col>
          <Col span={22}>
            <Input value={this.state.skill.name} onChange={e => {
              this.updateSkillField("name", e.target.value);
            }} />
          </Col>
        </Row>
        <Row style={{marginTop: "20px"}}>
          <Col style={{marginTop: "5px"}} span={(Setting.isMobile()) ? 22 : 2}>
            {Setting.getLabel(i18next.t("general:Display name"), i18next.t("general:Display name - Tooltip"))} :
          </Col>
          <Col span={22}>
            <Input value={this.state.skill.displayName} onChange={e => {
              this.updateSkillField("displayName", e.target.value);
            }} />
          </Col>
        </Row>
        <Row style={{marginTop: "20px"}}>
          <Col style={{marginTop: "5px"}} span={(Setting.isMobile()) ? 22 : 2}>
            {Setting.getLabel(i18next.t("general:Type"), i18next.t("general:Type - Tooltip"))} :
          </Col>
          <Col span={22}>
            <Select virtual={false} style={{width: "100%"}} value={this.state.skill.type}
              onChange={(value) => this.updateSkillField("type", value)}
            >
              {SKILL_TYPES.map((t, index) => (
                <Option key={index} value={t}>{t}</Option>
              ))}
            </Select>
          </Col>
        </Row>
        <Row style={{marginTop: "20px"}}>
          <Col style={{marginTop: "5px"}} span={(Setting.isMobile()) ? 22 : 2}>
            {Setting.getLabel(i18next.t("general:Description"), i18next.t("general:Description - Tooltip"))} :
          </Col>
          <Col span={22}>
            <Input value={this.state.skill.description} onChange={e => {
              this.updateSkillField("description", e.target.value);
            }} />
          </Col>
        </Row>
        <Row style={{marginTop: "20px"}}>
          <Col style={{marginTop: "5px"}} span={(Setting.isMobile()) ? 22 : 2}>
            {Setting.getLabel(i18next.t("skill:Content"), i18next.t("skill:Content - Tooltip"))} :
          </Col>
          <Col span={22}>
            <TextArea
              rows={12}
              value={this.state.skill.content}
              onChange={e => this.updateSkillField("content", e.target.value)}
              placeholder={i18next.t("skill:Content placeholder")}
            />
          </Col>
        </Row>
        <Row style={{marginTop: "20px"}}>
          <Col style={{marginTop: "5px"}} span={(Setting.isMobile()) ? 22 : 2}>
            {Setting.getLabel(i18next.t("general:State"), i18next.t("general:State - Tooltip"))} :
          </Col>
          <Col span={22}>
            <Select virtual={false} style={{width: "100%"}} value={this.state.skill.state}
              onChange={value => this.updateSkillField("state", value)}
              options={[
                {value: "Active", label: i18next.t("general:Active")},
                {value: "Inactive", label: i18next.t("general:Inactive")},
              ].map(item => Setting.getOption(item.label, item.value))} />
          </Col>
        </Row>
      </Card>
    );
  }

  submitSkillEdit(exitAfterSave) {
    const skill = Setting.deepCopy(this.state.skill);
    SkillBackend.updateSkill(this.state.skill.owner, this.state.skillName, skill)
      .then((res) => {
        if (res.status === "ok") {
          if (res.data) {
            Setting.showMessage("success", i18next.t("general:Successfully saved"));
            this.setState({
              skillName: this.state.skill.name,
              isNewSkill: false,
            });

            if (exitAfterSave) {
              this.props.history.push("/skills");
            } else {
              this.props.history.push(`/skills/${this.state.skill.name}`);
            }
          } else {
            Setting.showMessage("error", i18next.t("general:Failed to connect to server"));
            this.updateSkillField("name", this.state.skillName);
          }
        } else {
          Setting.showMessage("error", `${i18next.t("general:Failed to save")}: ${res.msg}`);
        }
      })
      .catch(error => {
        Setting.showMessage("error", `${i18next.t("general:Failed to save")}: ${error}`);
      });
  }

  cancelSkillEdit() {
    if (this.state.isNewSkill) {
      SkillBackend.deleteSkill(this.state.skill)
        .then((res) => {
          if (res.status === "ok") {
            Setting.showMessage("success", i18next.t("general:Cancelled successfully"));
            this.props.history.push("/skills");
          } else {
            Setting.showMessage("error", `${i18next.t("general:Failed to cancel")}: ${res.msg}`);
          }
        })
        .catch(error => {
          Setting.showMessage("error", `${i18next.t("general:Failed to cancel")}: ${error}`);
        });
    } else {
      this.props.history.push("/skills");
    }
  }

  render() {
    return (
      <div>
        {
          this.state.skill !== null ? this.renderSkill() : <Loading type="page" tip={i18next.t("general:Loading")} />
        }
        <div style={{marginTop: "20px", marginLeft: "40px"}}>
          <Button size="large" onClick={() => this.submitSkillEdit(false)}>{i18next.t("general:Save")}</Button>
          <Button style={{marginLeft: "20px"}} type="primary" size="large" onClick={() => this.submitSkillEdit(true)}>{i18next.t("general:Save & Exit")}</Button>
          {this.state.isNewSkill && <Button style={{marginLeft: "20px"}} size="large" onClick={() => this.cancelSkillEdit()}>{i18next.t("general:Cancel")}</Button>}
        </div>
      </div>
    );
  }
}

export default SkillEditPage;
