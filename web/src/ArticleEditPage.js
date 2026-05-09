// Copyright 2024 The OpenAgent Authors. All Rights Reserved.
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
import {Affix, Button, Card, Col, Input, Popover, Row, Select, Space} from "antd";
import * as ArticleBackend from "./backend/ArticleBackend";
import * as Setting from "./Setting";
import i18next from "i18next";
import * as WorkflowBackend from "./backend/WorkflowBackend";
import ArticleTable from "./table/ArticleTable";
import ArticleMenu from "./ArticleMenu";

const {Option} = Select;
const {TextArea} = Input;

class ArticleEditPage extends React.Component {
  constructor(props) {
    super(props);
    this.state = {
      classes: props,
      articleName: props.match.params.articleName,
      workflows: [],
      article: null,
      chatPageObj: null,
      loading: false,
      isNewArticle: props.location?.state?.isNewArticle || false,
    };

    this.articleTableRef = React.createRef();
  }

  UNSAFE_componentWillMount() {
    this.getArticle();
    this.getWorkflows();
  }

  componentDidMount() {
    document.addEventListener("keydown", this.handleKeyDown);
  }

  componentWillUnmount() {
    document.removeEventListener("keydown", this.handleKeyDown);
  }

  handleKeyDown = (event) => {
    if ((event.ctrlKey || event.metaKey) && event.key === "s") {
      event.preventDefault();
      this.submitArticleEdit(false);
    }
  };

  getArticle() {
    ArticleBackend.getArticle(this.props.account.name, this.state.articleName)
      .then((res) => {
        if (res.status === "ok") {
          this.setState({
            article: res.data,
          });
        } else {
          Setting.showMessage("error", `${i18next.t("general:Failed to get")}: ${res.msg}`);
        }
      });
  }

  getWorkflows() {
    WorkflowBackend.getWorkflows(this.props.account.name)
      .then((res) => {
        if (res.status === "ok") {
          this.setState({
            workflows: res.data,
          });
        } else {
          Setting.showMessage("error", `${i18next.t("general:Failed to get")}: ${res.msg}`);
        }
      });
  }

  parseArticleField(key, value) {
    if ([""].includes(key)) {
      value = Setting.myParseInt(value);
    }
    return value;
  }

  updateArticleField(key, value) {
    value = this.parseArticleField(key, value);

    const article = this.state.article;
    article[key] = value;
    this.setState({
      article: article,
    });
  }

  preprocessText(text) {
    text = text.split("\n").filter(line => !line.trim().startsWith("%")).join("\n");

    const ignoreCommands = [
      "\\documentclass", "\\usepackage", "\\setcounter", "\\urldef", "\\newcommand",
      "\\begin{document}", "\\end{document}", "\\titlerunning", "\\authorrunning",
      "\\toctitle", "\\tocauthor", "\\maketitle", "\\mainmatter", "\\bibliographystyle", "\\bibliography",
      "\\begin{spacing}", "\\end{spacing}", "\\noindent", "\\author", "\\institute", "\\email", "\\keywords", "\\label",
    ];
    text = text.split("\n").filter(line => {
      return !ignoreCommands.some(cmd => line.trim().startsWith(cmd));
    }).join("\n");

    text = text.replace("\\section*{", "\\section{");

    return text;
  }

  refineTextEn(text) {
    text = text.replace(/\n{3,}/g, "\n\n");
    text = text.replace(/^\n+/, "").replace(/\n+$/, "");
    return text;
  }

  splitTextBlocks(blocks) {
    const textMaxLength = 1000;

    let blockIndex = 0;
    blocks.forEach((block, index) => {
      if (block.type === "Text") {
        const paragraphs = block.textEn.split("\n");
        const newBlocks = [];
        let currentText = "";

        paragraphs.forEach((paragraph) => {
          if ((currentText.length + paragraph.length) > textMaxLength) {
            if (currentText.trim() !== "") {
              newBlocks.push({no: blockIndex++, type: "Text", text: "", textEn: currentText.trim(), state: ""});
              currentText = "";
            }
            if (paragraph.length > textMaxLength) {
              newBlocks.push({no: blockIndex++, type: "Text", text: "", textEn: paragraph, state: ""});
            } else {
              currentText = paragraph;
            }
          } else {
            currentText += paragraph + "\n";
          }
        });

        if (currentText.trim() !== "") {
          newBlocks.push({no: blockIndex++, type: "Text", text: "", textEn: currentText.trim(), state: ""});
        }

        blocks.splice(index, 1, ...newBlocks);
      }
    });

    return blocks;
  }

  parseText() {
    let text = this.state.article.text;
    text = this.preprocessText(text);
    this.updateArticleField("text", text);

    let blocks = [];
    let blockIndex = 0;

    const patterns = [
      {pattern: new RegExp("\\\\title\\{([^}]+)}", "g"), type: "Title"},
      {pattern: new RegExp("\\\\begin\\{abstract}([\\s\\S]*?)\\\\end\\{abstract}", "g"), type: "Abstract"},
      {pattern: new RegExp("\\\\section\\{([^}]+)}", "g"), type: "Header 1"},
      {pattern: new RegExp("\\\\subsection\\{([^}]+)}", "g"), type: "Header 2"},
      {pattern: new RegExp("\\\\subsubsection\\{([^}]+)}", "g"), type: "Header 3"},
    ];

    const matches = [];

    patterns.forEach(({pattern, type}) => {
      const allMatches = [...text.matchAll(pattern)];
      allMatches.forEach(match => {
        matches.push({
          index: match.index,
          length: match[0].length,
          type: type,
          text: match[1],
        });
      });
    });

    matches.sort((a, b) => a.index - b.index);

    let lastIndex = 0;
    matches.forEach(match => {
      if (match.index > lastIndex) {
        const textPart = text.substring(lastIndex, match.index).trim();
        if (textPart) {
          blocks.push({no: blockIndex++, type: "Text", text: "", textEn: textPart, state: ""});
        }
      }
      blocks.push({no: blockIndex++, type: match.type, text: "", textEn: match.text, state: ""});
      lastIndex = match.index + match.length;
    });

    if (lastIndex < text.length) {
      const textPart = text.substring(lastIndex).trim();
      if (textPart) {
        blocks.push({no: blockIndex++, type: "Text", text: "", textEn: textPart, state: ""});
      }
    }

    blocks.forEach(block => {
      block.textEn = this.refineTextEn(block.textEn);
    });

    blocks = this.splitTextBlocks(blocks);

    this.updateArticleField("content", blocks);
  }

  getBlocksWithPrefix(blocks) {
    let header1Counter = 0;
    let header2Counter = 0;
    let header3Counter = 0;
    let lastHeader1Index = 0;
    let lastHeader2Index = 0;

    return blocks.map((block) => {
      switch (block.type) {
      case "Title":
        block.prefix = "Tit: ";
        break;
      case "Abstract":
        block.prefix = "Abs: ";
        break;
      case "Text":
        block.prefix = "";
        break;
      case "Header 1":
        header1Counter++;
        header2Counter = 0;
        header3Counter = 0;
        lastHeader1Index = header1Counter;
        block.prefix = `${header1Counter}. `;
        break;
      case "Header 2":
        header2Counter++;
        header3Counter = 0;
        lastHeader2Index = header2Counter;
        block.prefix = `${lastHeader1Index}.${header2Counter} `;
        break;
      case "Header 3":
        header3Counter++;
        block.prefix = `${lastHeader1Index}.${lastHeader2Index}.${header3Counter} `;
        break;
      default:
        block.prefix = "";
      }
      return block;
    });
  }

  exportText(isEn) {
    const blocks = this.state.article.content;
    let text = "";

    blocks.forEach(block => {
      let blockText;
      if (isEn) {
        blockText = block.textEn;
        if (blockText === "") {
          blockText = block.text;
        }
      } else {
        blockText = block.text;
        if (blockText === "") {
          blockText = block.textEn;
        }
      }

      let label = `\\label{sec:${block.textEn}}`;
      if (label === "") {
        label = `\\label{sec:${block.text}}`;
      }

      switch (block.type) {
      case "Title":
        text += `\\title{${blockText}}\n\n`;
        break;
      case "Abstract":
        text += `\\begin{abstract}\n${blockText}\n\\end{abstract}\n\n`;
        break;
      case "Header 1":
        text += `\\section{${blockText}}\n${label}\n\n`;
        break;
      case "Header 2":
        text += `\\subsection{${blockText}}\n${label}\n\n`;
        break;
      case "Header 3":
        text += `\\subsubsection{${blockText}}\n${label}\n\n`;
        break;
      case "Text":
        text += `${blockText}\n\n`;
        break;
      default:
        Setting.showMessage("error", `${i18next.t("article:Unknown block type")}: ${block.type}`);
      }
    });

    this.updateArticleField("text", text);
  }

  renderArticleField(label, control, span = 8) {
    return (
      <Col style={{marginTop: "12px"}} span={Setting.isMobile() ? 22 : span}>
        {label && <div style={{marginBottom: "6px", color: "#595959", fontWeight: 500, lineHeight: "22px", fontSize: "13px"}}>{label}</div>}
        {control}
      </Col>
    );
  }

  renderArticleActions() {
    const btnStyle = {
      backgroundColor: "#F8F9FA",
      borderColor: "rgb(229, 229, 234)",
      border: "1px solid #E5E5EA",
      borderRadius: "10px",
      padding: "6px 10px",
    };
    return (
      <Space wrap>
        <Button style={btnStyle} onClick={() => this.submitArticleEdit(false)}>{i18next.t("general:Save")}</Button>
        <Button style={btnStyle} onClick={() => this.submitArticleEdit(true)}>{i18next.t("general:Save & Exit")}</Button>
        {this.state.isNewArticle && <Button style={btnStyle} onClick={() => this.cancelArticleEdit()}>{i18next.t("general:Cancel")}</Button>}
      </Space>
    );
  }

  renderArticle() {
    const article = this.state.article;
    const blocks = this.getBlocksWithPrefix(article.content);
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
          <span style={{fontSize: "22px", fontWeight: 600}}>{i18next.t("article:Edit Article")}</span>
          <div style={{display: "flex", gap: "8px", marginRight: "4px"}}>
            {this.renderArticleActions()}
          </div>
        </div>

        <Card size="small" title={renderCardTitle(i18next.t("general:General Settings"), i18next.t("general:General Settings desc"))} style={sectionCardStyle} headStyle={cardHeadStyle}>
          <Row gutter={rowGutter}>
            {this.renderArticleField(
              Setting.getLabel(i18next.t("general:Name"), i18next.t("general:Name - Tooltip")),
              <Input value={article.name} onChange={(e) => this.updateArticleField("name", e.target.value)} />,
              6
            )}
            {this.renderArticleField(
              Setting.getLabel(i18next.t("general:Display name"), i18next.t("general:Display name - Tooltip")),
              <Input value={article.displayName} onChange={(e) => this.updateArticleField("displayName", e.target.value)} />,
              6
            )}
            {this.props.account.name !== "admin" ? null : this.renderArticleField(
              Setting.getLabel(i18next.t("store:Workflow"), i18next.t("store:Workflow - Tooltip")),
              <Select virtual={false} style={{width: "100%"}} value={article.workflow} onChange={(value) => this.updateArticleField("workflow", value)}
                options={this.state.workflows.map((item) => Setting.getOption(`${item.displayName} (${item.name})`, `${item.name}`))} />,
              8
            )}
            {this.renderArticleField(
              Setting.getLabel(i18next.t("general:Text"), i18next.t("general:Text - Tooltip")),
              <Popover placement="left" content={
                <div style={{width: "1000px"}}>
                  <Select virtual={false} mode="tags" style={{width: "100%"}} value={article.glossary} onChange={(value) => this.updateArticleField("glossary", value)}>
                    {article.glossary?.map((item, index) => <Option key={index} value={item}>{item}</Option>)}
                  </Select>
                  <Button style={{marginTop: "20px", marginBottom: "20px", marginRight: "20px"}} onClick={() => this.parseText()}>{i18next.t("article:Parse")}</Button>
                  <Button style={{marginTop: "20px", marginBottom: "20px", marginRight: "20px"}} type="primary" onClick={() => this.exportText(true)}>{i18next.t("article:Export")}</Button>
                  <Button style={{marginTop: "20px", marginBottom: "20px"}} onClick={() => this.exportText(false)}>{i18next.t("article:Export ZH")}</Button>
                  <TextArea autoSize={{minRows: 1, maxRows: 30}} showCount value={article.text} onChange={(e) => this.updateArticleField("text", e.target.value)} />
                </div>
              } title="" trigger="hover">
                <Input value={article.text} onChange={(e) => this.updateArticleField("text", e.target.value)} />
              </Popover>,
              8
            )}
          </Row>
        </Card>

        <Card size="small" title={renderCardTitle(i18next.t("general:Content"), "")} style={sectionCardStyle} headStyle={cardHeadStyle}>
          <Row gutter={rowGutter}>
            <Col span={5}>
              <Affix offsetTop={0} style={{marginRight: "10px"}}>
                <div style={{height: "100vh", overflowY: "auto", borderRight: 0}}>
                  <ArticleMenu table={blocks} onGoToRow={(table, i) => {
                    if (this.articleTableRef.current) {
                      this.articleTableRef.current.goToRow(table, i);
                    }
                  }} />
                </div>
              </Affix>
            </Col>
            <Col span={19}>
              <ArticleTable ref={this.articleTableRef} article={article} table={blocks} onUpdateTable={(value) => this.updateArticleField("content", value)} onSubmitArticleEdit={() => this.submitArticleEdit(false)} />
            </Col>
          </Row>
        </Card>
      </div>
    );
  }

  submitArticleEdit(exitAfterSave) {
    const article = Setting.deepCopy(this.state.article);
    ArticleBackend.updateArticle(this.state.article.owner, this.state.articleName, article)
      .then((res) => {
        if (res.status === "ok") {
          if (res.data) {
            Setting.showMessage("success", i18next.t("general:Successfully saved"));
            this.setState({
              articleName: this.state.article.name,
              isNewArticle: false,
            });
            if (exitAfterSave) {
              this.props.history.push("/articles");
            } else {
              this.props.history.push(`/articles/${this.state.article.name}`);
            }
          } else {
            Setting.showMessage("error", i18next.t("general:Failed to save"));
            this.updateArticleField("name", this.state.articleName);
          }
        } else {
          Setting.showMessage("error", `${i18next.t("general:Failed to save")}: ${res.msg}`);
        }
      })
      .catch(error => {
        Setting.showMessage("error", `${i18next.t("general:Failed to save")}: ${error}`);
      });
  }

  cancelArticleEdit() {
    if (this.state.isNewArticle) {
      ArticleBackend.deleteArticle(this.state.article)
        .then((res) => {
          if (res.status === "ok") {
            Setting.showMessage("success", i18next.t("general:Cancelled successfully"));
            this.props.history.push("/articles");
          } else {
            Setting.showMessage("error", `${i18next.t("general:Failed to cancel")}: ${res.msg}`);
          }
        })
        .catch(error => {
          Setting.showMessage("error", `${i18next.t("general:Failed to cancel")}: ${error}`);
        });
    } else {
      this.props.history.push("/articles");
    }
  }

  render() {
    return (
      <div style={{background: "#F1F3F5", padding: "16px 20px 32px", minHeight: "100vh"}}>
        {this.state.article !== null ? this.renderArticle() : <Loading type="page" tip={i18next.t("general:Loading")} />}
      </div>
    );
  }
}

export default ArticleEditPage;
