package database

import (
	"database/sql"
	"time"
)

type Group struct {
	ID          int       `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	University  string    `json:"university,omitempty"`
	Degree      string    `json:"degree,omitempty"`
	CreatorID   int       `json:"creator_id"`
	InviteCode  string    `json:"invite_code"`
	CreatedAt   time.Time `json:"created_at"`
	MemberCount int       `json:"member_count"` // Computed field
}

func CreateGroup(name, description, university, degree string, creatorID int) (*Group, error) {
	code := generateRandomCode(8)
	query := `INSERT INTO groups (name, description, university, degree, creator_id, invite_code) VALUES (?, ?, ?, ?, ?, ?)`
	result, err := DB.Exec(query, name, description, university, degree, creatorID, code)
	if err != nil {
		return nil, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}

	// Auto-join creator as admin
	_, err = DB.Exec(`INSERT INTO group_members (group_id, user_id, role) VALUES (?, ?, 'admin')`, id, creatorID)
	if err != nil {
		// cleanup?
		return nil, err
	}

	return &Group{
		ID:          int(id),
		Name:        name,
		Description: description,
		University:  university,
		Degree:      degree,
		CreatorID:   creatorID,
		InviteCode:  code,
		CreatedAt:   time.Now(),
		MemberCount: 1,
	}, nil
}

func ListGroups(university, degree string) ([]Group, error) {
	var args []interface{}
	query := `
        SELECT g.id, g.name, g.description, g.university, g.degree, g.creator_id, g.created_at, g.invite_code,
        (SELECT COUNT(*) FROM group_members gm WHERE gm.group_id = g.id) as member_count
        FROM groups g
        WHERE 1=1
    `

	if university != "" {
		query += " AND g.university = ?"
		args = append(args, university)
	}
	if degree != "" {
		query += " AND g.degree = ?"
		args = append(args, degree)
	}

	query += " ORDER BY member_count DESC LIMIT 50"

	rows, err := DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var groups []Group
	for rows.Next() {
		var g Group
		var uni, deg, code sql.NullString
		if err := rows.Scan(&g.ID, &g.Name, &g.Description, &uni, &deg, &g.CreatorID, &g.CreatedAt, &code, &g.MemberCount); err != nil {
			return nil, err
		}
		g.University = uni.String
		g.Degree = deg.String
		g.InviteCode = code.String
		groups = append(groups, g)
	}

	return groups, nil
}

func JoinGroup(groupID, userID int) error {
	_, err := DB.Exec(`INSERT INTO group_members (group_id, user_id) VALUES (?, ?)`, groupID, userID)
	return err
}

func JoinGroupByCode(code string, userID int) (*Group, error) {
	var groupID int
	err := DB.QueryRow("SELECT id FROM groups WHERE invite_code = ?", code).Scan(&groupID)
	if err != nil {
		return nil, err
	}

	// Check membership
	isMember, _ := IsMember(groupID, userID)
	if !isMember {
		err = JoinGroup(groupID, userID)
		if err != nil {
			return nil, err
		}
	}

	// Return group info (simplified, could be full fetch)
	return &Group{ID: groupID}, nil
}

func IsMember(groupID, userID int) (bool, error) {
	var count int
	err := DB.QueryRow("SELECT COUNT(*) FROM group_members WHERE group_id = ? AND user_id = ?", groupID, userID).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// GroupWithMembership includes is_member flag for the current user
type GroupWithMembership struct {
	ID          int       `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	University  string    `json:"university,omitempty"`
	Degree      string    `json:"degree,omitempty"`
	CreatorID   int       `json:"creator_id"`
	InviteCode  string    `json:"invite_code"`
	CreatedAt   time.Time `json:"created_at"`
	MemberCount int       `json:"member_count"`
	IsMember    bool      `json:"is_member"`
}

func ListGroupsWithMembership(university, degree string, userID int) ([]GroupWithMembership, error) {
	var args []interface{}
	query := `
        SELECT g.id, g.name, g.description, g.university, g.degree, g.creator_id, g.created_at, g.invite_code,
        (SELECT COUNT(*) FROM group_members gm WHERE gm.group_id = g.id) as member_count,
        CASE WHEN EXISTS(SELECT 1 FROM group_members gm WHERE gm.group_id = g.id AND gm.user_id = ?) THEN 1 ELSE 0 END as is_member
        FROM groups g
        WHERE 1=1
    `
	args = append(args, userID)

	if university != "" {
		query += " AND g.university = ?"
		args = append(args, university)
	}
	if degree != "" {
		query += " AND g.degree = ?"
		args = append(args, degree)
	}

	query += " ORDER BY member_count DESC LIMIT 50"

	rows, err := DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var groups []GroupWithMembership
	for rows.Next() {
		var g GroupWithMembership
		var uni, deg, code sql.NullString
		var isMemberInt int
		if err := rows.Scan(&g.ID, &g.Name, &g.Description, &uni, &deg, &g.CreatorID, &g.CreatedAt, &code, &g.MemberCount, &isMemberInt); err != nil {
			continue
		}
		g.University = uni.String
		g.Degree = deg.String
		g.InviteCode = code.String
		g.IsMember = isMemberInt == 1
		groups = append(groups, g)
	}

	return groups, nil
}
