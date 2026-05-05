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
	"fmt"
	"strings"

	"github.com/the-open-agent/openagent/util"
	"xorm.io/core"
)

type Skill struct {
	Owner       string `xorm:"varchar(100) notnull pk" json:"owner"`
	Name        string `xorm:"varchar(100) notnull pk" json:"name"`
	CreatedTime string `xorm:"varchar(100)" json:"createdTime"`

	DisplayName string `xorm:"varchar(100)" json:"displayName"`
	Type        string `xorm:"varchar(100)" json:"type"`
	Description string `xorm:"varchar(500)" json:"description"`
	Content     string `xorm:"mediumtext" json:"content"`

	State string `xorm:"varchar(100)" json:"state"`
}

func (s *Skill) GetId() string {
	return fmt.Sprintf("%s/%s", s.Owner, s.Name)
}

func GetGlobalSkills() ([]*Skill, error) {
	skills := []*Skill{}
	err := adapter.engine.Asc("owner").Desc("created_time").Find(&skills)
	return skills, err
}

func GetSkills(owner string) ([]*Skill, error) {
	skills := []*Skill{}
	err := adapter.engine.Desc("created_time").Find(&skills, &Skill{Owner: owner})
	return skills, err
}

func getSkill(owner string, name string) (*Skill, error) {
	s := Skill{Owner: owner, Name: name}
	existed, err := adapter.engine.Get(&s)
	if err != nil {
		return &s, err
	}
	if existed {
		return &s, nil
	}
	return nil, nil
}

func GetSkill(id string) (*Skill, error) {
	owner, name, err := util.GetOwnerAndNameFromIdWithError(id)
	if err != nil {
		return nil, err
	}
	return getSkill(owner, name)
}

func GetSkillByOwnerAndName(owner string, nameOrId string) (*Skill, error) {
	if nameOrId == "" {
		return nil, nil
	}
	var id string
	if _, _, err := util.GetOwnerAndNameFromIdWithError(nameOrId); err == nil {
		id = nameOrId
	} else {
		id = util.GetIdFromOwnerAndName(owner, nameOrId)
	}
	s, err := GetSkill(id)
	if err != nil {
		return nil, err
	}
	if s != nil {
		return s, nil
	}
	if owner != "admin" && !strings.Contains(nameOrId, "/") {
		return GetSkill(util.GetIdFromOwnerAndName("admin", nameOrId))
	}
	return nil, nil
}

func GetSkillCount(owner, field, value string) (int64, error) {
	session := GetDbSession(owner, -1, -1, field, value, "", "")
	return session.Count(&Skill{})
}

func GetPaginationSkills(owner string, offset, limit int, field, value, sortField, sortOrder string) ([]*Skill, error) {
	skills := []*Skill{}
	session := GetDbSession(owner, offset, limit, field, value, sortField, sortOrder)
	err := session.Find(&skills)
	return skills, err
}

func UpdateSkill(id string, s *Skill) (bool, error) {
	owner, name, err := util.GetOwnerAndNameFromIdWithError(id)
	if err != nil {
		return false, err
	}
	skillDb, err := getSkill(owner, name)
	if err != nil {
		return false, err
	}
	if s == nil || skillDb == nil {
		return false, nil
	}

	_, err = adapter.engine.ID(core.PK{owner, name}).AllCols().Update(s)
	if err != nil {
		return false, err
	}
	return true, nil
}

func AddSkill(s *Skill) (bool, error) {
	affected, err := adapter.engine.Insert(s)
	if err != nil {
		return false, err
	}
	return affected != 0, nil
}

func DeleteSkill(s *Skill) (bool, error) {
	affected, err := adapter.engine.ID(core.PK{s.Owner, s.Name}).Delete(&Skill{})
	if err != nil {
		return false, err
	}
	return affected != 0, nil
}

// GetSkillsContent loads all named skills for a store and concatenates their content.
// Returns an empty string if no skills are configured or none are active.
func GetSkillsContent(owner string, skillNames []string) (string, error) {
	if len(skillNames) == 0 {
		return "", nil
	}

	var parts []string
	for _, name := range skillNames {
		s, err := GetSkillByOwnerAndName(owner, name)
		if err != nil {
			return "", err
		}
		if s == nil || s.State != "Active" || strings.TrimSpace(s.Content) == "" {
			continue
		}
		parts = append(parts, strings.TrimSpace(s.Content))
	}

	return strings.Join(parts, "\n\n"), nil
}
