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

package object

import (
	"github.com/the-open-agent/openagent/util"
	"xorm.io/core"
)

type Site struct {
	Owner       string `xorm:"varchar(100) notnull pk" json:"owner"`
	Name        string `xorm:"varchar(100) notnull pk" json:"name"`
	CreatedTime string `xorm:"varchar(100)" json:"createdTime"`
	DisplayName string `xorm:"varchar(100)" json:"displayName"`

	ThemeColor       string   `xorm:"varchar(100)" json:"themeColor"`
	HtmlTitle        string   `xorm:"varchar(100)" json:"htmlTitle"`
	FaviconUrl       string   `xorm:"varchar(200)" json:"faviconUrl"`
	LogoUrl          string   `xorm:"varchar(200)" json:"logoUrl"`
	FooterHtml       string   `xorm:"mediumtext" json:"footerHtml"`
	NavItems         []string `xorm:"text" json:"navItems"`
	CheckUserBalance bool     `xorm:"bool" json:"checkUserBalance"`
}

func GetGlobalSites() ([]*Site, error) {
	sites := []*Site{}
	err := adapter.engine.Asc("owner").Desc("created_time").Find(&sites)
	if err != nil {
		return nil, err
	}
	return sites, nil
}

func GetSites(owner string) ([]*Site, error) {
	sites := []*Site{}
	err := adapter.engine.Desc("created_time").Where("owner = ?", owner).Find(&sites)
	if err != nil {
		return nil, err
	}
	return sites, nil
}

func GetSite(id string) (*Site, error) {
	owner, name, err := util.GetOwnerAndNameFromIdWithError(id)
	if err != nil {
		return nil, err
	}
	site := &Site{Owner: owner, Name: name}
	existed, err := adapter.engine.Get(site)
	if err != nil {
		return nil, err
	}
	if !existed {
		return nil, nil
	}
	return site, nil
}

func GetBuiltInSite() (*Site, error) {
	return GetSite("admin/site-built-in")
}

func UpdateSite(id string, site *Site) (bool, error) {
	owner, name, err := util.GetOwnerAndNameFromIdWithError(id)
	if err != nil {
		return false, err
	}
	if s, err := GetSite(id); err != nil {
		return false, err
	} else if s == nil {
		return false, nil
	}
	affected, err2 := adapter.engine.ID(core.PK{owner, name}).AllCols().Update(site)
	if err2 != nil {
		return false, err2
	}
	return affected != 0, nil
}

func AddSite(site *Site) (bool, error) {
	affected, err := adapter.engine.Insert(site)
	if err != nil {
		return false, err
	}
	return affected != 0, nil
}

func DeleteSite(site *Site) (bool, error) {
	affected, err := adapter.engine.ID(core.PK{site.Owner, site.Name}).Delete(&Site{})
	if err != nil {
		return false, err
	}
	return affected != 0, nil
}
