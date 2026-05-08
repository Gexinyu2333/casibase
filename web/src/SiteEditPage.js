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
import {Button, Card, Col, Image, Input, Popover, Row, Space, Switch, Upload} from "antd";
import * as SiteBackend from "./backend/SiteBackend";
import * as ResourceBackend from "./backend/ResourceBackend";
import * as Setting from "./Setting";
import {ThemeDefault} from "./Conf";
import i18next from "i18next";
import {LinkOutlined, UploadOutlined} from "@ant-design/icons";
import Editor from "./common/Editor";
import {NavItemTree} from "./component/nav-item-tree/NavItemTree";

class SiteEditPage extends React.Component {
  constructor(props) {
    super(props);
    this.state = {
      classes: props,
      siteName: props.match.params.siteName,
      site: null,
      uploadingFavicon: false,
      uploadingLogo: false,
    };
  }

  UNSAFE_componentWillMount() {
    this.getSite();
  }

  getSite() {
    SiteBackend.getSite("admin", this.state.siteName)
      .then((res) => {
        if (res.status === "ok") {
          this.setState({site: res.data});
        } else {
          Setting.showMessage("error", `${i18next.t("general:Failed to get")}: ${res.msg}`);
        }
      });
  }

  updateSiteField(key, value) {
    const site = Setting.deepCopy(this.state.site);
    site[key] = value;
    this.setState({site});
  }

  handleImageUpload(field, file) {
    const loadingKey = field === "faviconUrl" ? "uploadingFavicon" : "uploadingLogo";
    this.setState({[loadingKey]: true});
    ResourceBackend.uploadResource("admin", "avatar", "site", this.state.site.name, file)
      .then((res) => {
        if (res.status === "ok") {
          this.updateSiteField(field, res.data);
          Setting.showMessage("success", i18next.t("general:Successfully uploaded"));
        } else {
          Setting.showMessage("error", `${i18next.t("general:Failed to upload")}: ${res.msg}`);
        }
      })
      .catch(err => {
        Setting.showMessage("error", `${i18next.t("general:Failed to upload")}: ${err.message}`);
      })
      .finally(() => {
        this.setState({[loadingKey]: false});
      });
  }

  submitSiteEdit(exitAfterSave) {
    SiteBackend.updateSite(this.state.site.owner, this.state.siteName, this.state.site)
      .then((res) => {
        if (res.status === "ok") {
          if (res.data) {
            Setting.showMessage("success", i18next.t("general:Successfully saved"));
            Setting.setThemeColor(this.state.site.themeColor || Setting.getThemeColor());
            this.setState({siteName: this.state.site.name});
            if (exitAfterSave) {
              this.props.history.push(`/sites/${this.state.site.name}`);
            } else {
              this.props.history.push(`/sites/${this.state.site.name}`);
            }
          } else {
            Setting.showMessage("error", i18next.t("general:Failed to connect to server"));
            this.updateSiteField("name", this.state.siteName);
          }
        } else {
          Setting.showMessage("error", `${i18next.t("general:Failed to save")}: ${res.msg}`);
        }
      })
      .catch(error => {
        Setting.showMessage("error", `${i18next.t("general:Failed to save")}: ${error}`);
      });
  }

  renderSite() {
    return (
      <Card size="small" title={
        <div>
          {i18next.t("site:Edit Site")}&nbsp;&nbsp;&nbsp;&nbsp;
          <Button onClick={() => this.submitSiteEdit(false)}>{i18next.t("general:Save")}</Button>
          <Button style={{marginLeft: "20px"}} type="primary" onClick={() => this.submitSiteEdit(true)}>{i18next.t("general:Save & Exit")}</Button>
        </div>
      } style={{marginLeft: "5px"}} type="inner">
        <Row style={{marginTop: "10px"}} >
          <Col style={{marginTop: "5px"}} span={(Setting.isMobile()) ? 22 : 2}>
            {Setting.getLabel(i18next.t("general:Name"), i18next.t("general:Name - Tooltip"))} :
          </Col>
          <Col span={22} >
            <Input value={this.state.site.name} disabled={this.state.site.name === "site-built-in"} onChange={e => {
              this.updateSiteField("name", e.target.value);
            }} />
          </Col>
        </Row>
        <Row style={{marginTop: "20px"}} >
          <Col style={{marginTop: "5px"}} span={(Setting.isMobile()) ? 22 : 2}>
            {Setting.getLabel(i18next.t("general:Display name"), i18next.t("general:Display name - Tooltip"))} :
          </Col>
          <Col span={22} >
            <Input value={this.state.site.displayName} onChange={e => {
              this.updateSiteField("displayName", e.target.value);
            }} />
          </Col>
        </Row>
        <Row style={{marginTop: "20px"}} >
          <Col style={{marginTop: "5px"}} span={(Setting.isMobile()) ? 22 : 2}>
            {Setting.getLabel(i18next.t("store:Theme color"), i18next.t("store:Theme color - Tooltip"))} :
          </Col>
          <Col span={22} >
            <input type="color" value={this.state.site.themeColor || ThemeDefault.colorPrimary} onChange={(e) => {
              this.updateSiteField("themeColor", e.target.value);
            }} />
          </Col>
        </Row>
        <Row style={{marginTop: "20px"}} >
          <Col style={{marginTop: "5px"}} span={(Setting.isMobile()) ? 22 : 2}>
            {Setting.getLabel(i18next.t("general:HTML title"), i18next.t("general:HTML title - Tooltip"))} :
          </Col>
          <Col span={22} >
            <Input value={this.state.site.htmlTitle} onChange={e => {
              this.updateSiteField("htmlTitle", e.target.value);
            }} />
          </Col>
        </Row>
        <Row style={{marginTop: "20px"}} >
          <Col style={{marginTop: "5px"}} span={(Setting.isMobile()) ? 22 : 2}>
            {Setting.getLabel(i18next.t("general:Favicon URL"), i18next.t("general:Favicon URL - Tooltip"))} :
          </Col>
          <Col span={22} >
            <Space direction="vertical" style={{width: "100%"}}>
              <Space.Compact style={{width: "100%"}}>
                <Input prefix={<LinkOutlined />} value={this.state.site.faviconUrl} onChange={e => {
                  this.updateSiteField("faviconUrl", e.target.value);
                }} />
                <Upload name="file" accept="image/*" showUploadList={false} customRequest={({file}) => this.handleImageUpload("faviconUrl", file)}>
                  <Button icon={<UploadOutlined />} loading={this.state.uploadingFavicon}>
                    {i18next.t("general:Upload")}
                  </Button>
                </Upload>
              </Space.Compact>
              {this.state.site.faviconUrl ? (
                <Image src={Setting.getFaviconUrl("", this.state.site.faviconUrl)} alt={this.state.site.faviconUrl} height={90}
                  preview={{mask: i18next.t("general:Preview")}}
                />
              ) : null}
            </Space>
          </Col>
        </Row>
        <Row style={{marginTop: "20px"}} >
          <Col style={{marginTop: "5px"}} span={(Setting.isMobile()) ? 22 : 2}>
            {Setting.getLabel(i18next.t("general:Logo URL"), i18next.t("general:Logo URL - Tooltip"))} :
          </Col>
          <Col span={22} >
            <Space direction="vertical" style={{width: "100%"}}>
              <Space.Compact style={{width: "100%"}}>
                <Input prefix={<LinkOutlined />} value={this.state.site.logoUrl} onChange={e => {
                  this.updateSiteField("logoUrl", e.target.value);
                }} />
                <Upload name="file" accept="image/*" showUploadList={false} customRequest={({file}) => this.handleImageUpload("logoUrl", file)}>
                  <Button icon={<UploadOutlined />} loading={this.state.uploadingLogo}>
                    {i18next.t("general:Upload")}
                  </Button>
                </Upload>
              </Space.Compact>
              {this.state.site.logoUrl ? (
                <Image src={Setting.getLogo("", this.state.site.logoUrl)} alt={this.state.site.logoUrl} height={90}
                  preview={{mask: i18next.t("general:Preview")}}
                />
              ) : null}
            </Space>
          </Col>
        </Row>
        <Row style={{marginTop: "20px"}} >
          <Col style={{marginTop: "5px"}} span={(Setting.isMobile()) ? 22 : 2}>
            {Setting.getLabel(i18next.t("general:Footer HTML"), i18next.t("general:Footer HTML - Tooltip"))} :
          </Col>
          <Col span={22} >
            <Popover placement="right" content={
              <div style={{width: "900px", height: "300px"}} >
                <Editor
                  value={this.state.site.footerHtml}
                  lang="html"
                  fillHeight
                  dark
                  onChange={value => {
                    this.updateSiteField("footerHtml", value);
                  }}
                />
              </div>
            } title={i18next.t("store:Footer HTML - Edit")} trigger="click">
              <Input value={this.state.site.footerHtml} style={{marginBottom: "10px"}} onChange={e => {
                this.updateSiteField("footerHtml", e.target.value);
              }} />
            </Popover>
          </Col>
        </Row>
        <Row style={{marginTop: "20px"}} >
          <Col style={{marginTop: "5px"}} span={(Setting.isMobile()) ? 22 : 2}>
            {Setting.getLabel(i18next.t("store:Navbar items"), i18next.t("store:Navbar items - Tooltip"))} :
          </Col>
          <Col span={22} >
            <NavItemTree
              disabled={!Setting.isAdminUser(this.props.account)}
              checkedKeys={this.state.site.navItems ?? ["all"]}
              defaultExpandedKeys={["all"]}
              onCheck={(checked) => {
                this.updateSiteField("navItems", checked);
              }}
            />
          </Col>
        </Row>
        <Row style={{marginTop: "20px"}} >
          <Col style={{marginTop: "5px"}} span={(Setting.isMobile()) ? 22 : 2}>
            {Setting.getLabel(i18next.t("site:Check user balance"), i18next.t("site:Check user balance - Tooltip"))} :
          </Col>
          <Col span={22} >
            <Switch checked={this.state.site.checkUserBalance} onChange={checked => {
              this.updateSiteField("checkUserBalance", checked);
            }} />
          </Col>
        </Row>
      </Card>
    );
  }

  render() {
    return (
      <div>
        {this.state.site !== null ? this.renderSite() : <Loading type="page" tip={i18next.t("general:Loading")} />}
        <div style={{marginTop: "20px", marginLeft: "40px"}}>
          <Button size="large" onClick={() => this.submitSiteEdit(false)}>{i18next.t("general:Save")}</Button>
          <Button style={{marginLeft: "20px"}} type="primary" size="large" onClick={() => this.submitSiteEdit(true)}>{i18next.t("general:Save & Exit")}</Button>
        </div>
      </div>
    );
  }
}

export default SiteEditPage;
