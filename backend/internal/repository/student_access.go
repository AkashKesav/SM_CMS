package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"cms-backend/internal/model"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var studentOwnedTables = map[string]bool{
	"students":                true,
	"achievements":            true,
	"achievement_members":     true,
	"student_tags":            true,
	"projects":                true,
	"project_members":         true,
	"hof_competition_members": true,
	"hof_coordinators":        true,
	"hof_internships":         true,
}

var initialAdminRollNos = map[string]bool{
	"23bsm006": true,
	"23bsm007": true,
}

type StudentAccessRepository struct {
	db *DBWrapper
}

func NewStudentAccessRepository(db *DBWrapper) *StudentAccessRepository {
	return &StudentAccessRepository{db: db}
}

func EnsureStudentAccessSchema(pool *pgxpool.Pool) error {
	if pool == nil {
		return nil
	}

	sql := `
ALTER TABLE public.profiles DROP CONSTRAINT IF EXISTS profiles_role_check;
ALTER TABLE public.profiles ADD CONSTRAINT profiles_role_check CHECK (role IN ('admin', 'student'));
ALTER TABLE public.profiles ADD COLUMN IF NOT EXISTS student_id uuid REFERENCES public.students(id) ON DELETE SET NULL;
CREATE INDEX IF NOT EXISTS idx_profiles_student_id ON public.profiles(student_id);

CREATE TABLE IF NOT EXISTS public.student_change_requests (
	id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
	requester_user_id uuid NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
	student_id uuid REFERENCES public.students(id) ON DELETE SET NULL,
	request_type text NOT NULL CHECK (request_type IN ('claim_student', 'create', 'update', 'delete', 'link', 'unlink')),
	target_table text NOT NULL CHECK (target_table IN ('students', 'achievements', 'achievement_members', 'student_tags', 'projects', 'project_members', 'hof_competition_members', 'hof_coordinators', 'hof_internships')),
	target_key jsonb,
	current_data jsonb,
	proposed_data jsonb NOT NULL DEFAULT '{}'::jsonb,
	status text NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'approved', 'rejected')),
	admin_note text,
	reviewed_by uuid REFERENCES auth.users(id) ON DELETE SET NULL,
	reviewed_at timestamptz,
	created_at timestamptz NOT NULL DEFAULT now(),
	updated_at timestamptz NOT NULL DEFAULT now()
);
ALTER TABLE public.student_change_requests DROP CONSTRAINT IF EXISTS student_change_requests_target_table_check;
ALTER TABLE public.student_change_requests
	ADD CONSTRAINT student_change_requests_target_table_check
	CHECK (target_table IN ('students', 'achievements', 'achievement_members', 'student_tags', 'projects', 'project_members', 'hof_competition_members', 'hof_coordinators', 'hof_internships'));
CREATE INDEX IF NOT EXISTS idx_student_change_requests_requester ON public.student_change_requests(requester_user_id);
CREATE INDEX IF NOT EXISTS idx_student_change_requests_student ON public.student_change_requests(student_id);
CREATE INDEX IF NOT EXISTS idx_student_change_requests_status ON public.student_change_requests(status, created_at DESC);
ALTER TABLE public.student_change_requests ENABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS student_read_own_change_requests ON public.student_change_requests;
CREATE POLICY student_read_own_change_requests
	ON public.student_change_requests
	FOR SELECT
	USING (requester_user_id = auth.uid());

DROP POLICY IF EXISTS student_create_own_change_requests ON public.student_change_requests;
CREATE POLICY student_create_own_change_requests
	ON public.student_change_requests
	FOR INSERT
	WITH CHECK (requester_user_id = auth.uid());

DROP POLICY IF EXISTS admin_manage_student_change_requests ON public.student_change_requests;
CREATE POLICY admin_manage_student_change_requests
	ON public.student_change_requests
	USING (public.is_admin())
	WITH CHECK (public.is_admin());

DROP POLICY IF EXISTS admin_manage_profiles ON public.profiles;
CREATE POLICY admin_manage_profiles
	ON public.profiles
	USING (public.is_admin())
	WITH CHECK (public.is_admin());

DROP POLICY IF EXISTS user_can_read_own_profile ON public.profiles;
CREATE POLICY user_can_read_own_profile
	ON public.profiles
	FOR SELECT
	USING (id = auth.uid());

DROP POLICY IF EXISTS user_can_read_linked_student ON public.students;
CREATE POLICY user_can_read_linked_student
	ON public.students
	FOR SELECT
	USING (
		id IN (
			SELECT student_id
			FROM public.profiles
			WHERE profiles.id = auth.uid()
		)
	);
`
	_, err := pool.Exec(context.Background(), sql)
	return err
}

func IsStudentOwnedTable(table string) bool {
	return studentOwnedTables[table]
}

func GeneratedStudentEmail(rollNo string) string {
	rollNo = strings.ToLower(strings.TrimSpace(rollNo))
	if rollNo == "" {
		return ""
	}
	return rollNo + "@iiitdmj.ac.in"
}

func InitialRoleForRollNo(rollNo string) string {
	if initialAdminRollNos[strings.ToLower(strings.TrimSpace(rollNo))] {
		return "admin"
	}
	return "student"
}

func (r *StudentAccessRepository) GetProfile(ctx context.Context, userID, email string) (model.UserProfile, error) {
	profile := model.UserProfile{
		ID:    userID,
		Email: email,
		Role:  "student",
	}

	if userID == "" {
		return profile, fmt.Errorf("missing user id")
	}
	if strings.HasPrefix(userID, "demo-") {
		profile.Role = "admin"
		return profile, nil
	}
	if r.db.IsInMemory() {
		profile.Role = "admin"
		return profile, nil
	}

	var studentID *string
	err := r.db.Pool().QueryRow(ctx, `
		SELECT role, student_id::text
		FROM public.profiles
		WHERE id = $1
	`, userID).Scan(&profile.Role, &studentID)
	if err != nil {
		if err == pgx.ErrNoRows {
			profile.Role = "student"
			return profile, nil
		}
		return profile, err
	}
	profile.StudentID = studentID
	return profile, nil
}

func (r *StudentAccessRepository) UpsertProfile(ctx context.Context, userID, role string, studentID *string) error {
	if r.db.IsInMemory() {
		return nil
	}
	if role == "" {
		role = "student"
	}
	_, err := r.db.Pool().Exec(ctx, `
		INSERT INTO public.profiles (id, role, student_id)
		VALUES ($1, $2, $3)
		ON CONFLICT (id) DO UPDATE
		SET role = EXCLUDED.role,
		    student_id = COALESCE(public.profiles.student_id, EXCLUDED.student_id)
	`, userID, role, studentID)
	return err
}

func (r *StudentAccessRepository) SetProfileForStudent(ctx context.Context, userID, role, studentID string) error {
	if r.db.IsInMemory() {
		return nil
	}
	if userID == "" || studentID == "" {
		return fmt.Errorf("user id and student id are required")
	}
	if role != "admin" {
		role = "student"
	}
	_, err := r.db.Pool().Exec(ctx, `
		INSERT INTO public.profiles (id, role, student_id)
		VALUES ($1, $2, $3)
		ON CONFLICT (id) DO UPDATE
		SET role = EXCLUDED.role,
		    student_id = EXCLUDED.student_id
	`, userID, role, studentID)
	return err
}

func (r *StudentAccessRepository) SetUserRole(ctx context.Context, userID, role string) error {
	if r.db.IsInMemory() {
		return nil
	}
	if userID == "" {
		return fmt.Errorf("user id is required")
	}
	if role != "admin" {
		role = "student"
	}
	result, err := r.db.Pool().Exec(ctx, `
		UPDATE public.profiles
		SET role = $2
		WHERE id = $1
	`, userID, role)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("profile not found")
	}
	return nil
}

func (r *StudentAccessRepository) NormalizeProfileRoles(ctx context.Context) error {
	if r.db.IsInMemory() {
		return nil
	}

	_, err := r.db.Pool().Exec(ctx, `
		UPDATE public.profiles p
		SET role = CASE
			WHEN lower(s.roll_no) IN ('23bsm006', '23bsm007') THEN 'admin'
			ELSE 'student'
		END
		FROM public.students s
		WHERE p.student_id = s.id
	`)
	if err != nil {
		return err
	}

	_, err = r.db.Pool().Exec(ctx, `
		UPDATE public.profiles
		SET role = 'student'
		WHERE student_id IS NULL
	`)
	return err
}

func (r *StudentAccessRepository) ResolveLoginIdentifier(ctx context.Context, identifier string) (string, error) {
	identifier = strings.TrimSpace(identifier)
	if identifier == "" {
		return "", fmt.Errorf("login identifier is required")
	}
	if strings.Contains(identifier, "@") || r.db.IsInMemory() {
		return identifier, nil
	}

	var email string
	err := r.db.Pool().QueryRow(ctx, `
		SELECT COALESCE(NULLIF(au.email, ''), NULLIF(s.email, ''), lower(s.roll_no) || '@iiitdmj.ac.in')
		FROM public.students s
		LEFT JOIN public.profiles p ON p.student_id = s.id
		LEFT JOIN auth.users au ON au.id = p.id
		WHERE lower(s.roll_no) = lower($1)
		ORDER BY au.email IS NULL, p.id IS NULL
		LIMIT 1
	`, identifier).Scan(&email)
	if err != nil {
		if err == pgx.ErrNoRows {
			return identifier, nil
		}
		return "", err
	}
	return email, nil
}

func (r *StudentAccessRepository) GetAuthUserIDByEmail(ctx context.Context, email string) (string, error) {
	if r.db.IsInMemory() {
		return "", pgx.ErrNoRows
	}
	var userID string
	err := r.db.Pool().QueryRow(ctx, `
		SELECT id::text
		FROM auth.users
		WHERE lower(email) = lower($1)
		LIMIT 1
	`, email).Scan(&userID)
	return userID, err
}

func (r *StudentAccessRepository) ListAccessUsers(ctx context.Context) ([]model.AccessUser, error) {
	if r.db.IsInMemory() {
		return []model.AccessUser{
			{
				UserID:     "demo-user-id",
				StudentID:  "demo-student-id",
				RollNo:     "23bsm006",
				Name:       "Demo Admin",
				LoginEmail: "demo@example.com",
				Role:       "admin",
				HasAccount: true,
			},
		}, nil
	}

	rows, err := r.db.Pool().Query(ctx, `
		SELECT
			s.id::text,
			COALESCE(s.roll_no, '') AS roll_no,
			COALESCE(s.name, '') AS name,
			COALESCE(
				NULLIF(au.email, ''),
				NULLIF(s.email, ''),
				lower(s.roll_no) || '@iiitdmj.ac.in'
			) AS login_email,
			COALESCE(p.id::text, '') AS user_id,
			COALESCE(p.role, 'student') AS role,
			(au.id IS NOT NULL) AS has_account
		FROM public.students s
		LEFT JOIN LATERAL (
			SELECT p.id, p.role
			FROM public.profiles p
			WHERE p.student_id = s.id
			ORDER BY p.id
			LIMIT 1
		) p ON true
		LEFT JOIN auth.users au ON au.id = p.id
		ORDER BY lower(COALESCE(s.roll_no, '')), lower(COALESCE(s.name, ''))
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	users := make([]model.AccessUser, 0)
	for rows.Next() {
		var user model.AccessUser
		if err := rows.Scan(
			&user.StudentID,
			&user.RollNo,
			&user.Name,
			&user.LoginEmail,
			&user.UserID,
			&user.Role,
			&user.HasAccount,
		); err != nil {
			return nil, err
		}
		if user.UserID == "" || user.Role == "" {
			user.Role = InitialRoleForRollNo(user.RollNo)
		}
		if user.LoginEmail == "" {
			user.LoginEmail = GeneratedStudentEmail(user.RollNo)
		}
		users = append(users, user)
	}
	return users, rows.Err()
}

func (r *StudentAccessRepository) CreateRegistrationRequest(ctx context.Context, userID, email, name, rollNo string) (model.StudentChangeRequest, error) {
	if r.db.IsInMemory() {
		return model.StudentChangeRequest{}, fmt.Errorf("student registration requests require a database")
	}
	rollNo = strings.TrimSpace(rollNo)
	if rollNo == "" {
		return model.StudentChangeRequest{}, fmt.Errorf("roll number is required")
	}

	if err := r.UpsertProfile(ctx, userID, "student", nil); err != nil {
		return model.StudentChangeRequest{}, err
	}

	student, err := r.FindStudentByRollNo(ctx, rollNo)
	if err != nil {
		return model.StudentChangeRequest{}, err
	}

	proposed := map[string]interface{}{
		"roll_no": rollNo,
		"email":   email,
	}
	if name != "" {
		proposed["name"] = name
	}

	request := model.StudentChangeRequest{
		RequesterUserID: userID,
		RequestType:     "create",
		TargetTable:     "students",
		ProposedData:    proposed,
		Status:          "pending",
	}

	if student != nil {
		studentID := fmt.Sprintf("%v", student["id"])
		claimed, err := r.IsStudentClaimed(ctx, studentID, userID)
		if err != nil {
			return model.StudentChangeRequest{}, err
		}
		if claimed {
			return model.StudentChangeRequest{}, fmt.Errorf("this roll number is already linked to another account")
		}
		request.StudentID = &studentID
		request.RequestType = "claim_student"
		request.TargetKey = map[string]interface{}{"id": studentID}
		request.CurrentData = student
	}

	return r.CreateChangeRequest(ctx, request)
}

func (r *StudentAccessRepository) FindStudentByRollNo(ctx context.Context, rollNo string) (map[string]interface{}, error) {
	rows, err := r.db.Pool().Query(ctx, `
		SELECT *
		FROM public.students
		WHERE lower(roll_no) = lower($1)
		LIMIT 1
	`, rollNo)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanOneRecord(rows)
}

func (r *StudentAccessRepository) IsStudentClaimed(ctx context.Context, studentID, excludeUserID string) (bool, error) {
	var count int
	err := r.db.Pool().QueryRow(ctx, `
		SELECT COUNT(*)
		FROM public.profiles
		WHERE student_id = $1
		  AND id::text <> $2
	`, studentID, excludeUserID).Scan(&count)
	return count > 0, err
}

func (r *StudentAccessRepository) GetWorkspace(ctx context.Context, profile model.UserProfile) (model.StudentWorkspace, error) {
	workspace := model.StudentWorkspace{
		Profile:       profile,
		LinkedRecords: map[string][]map[string]interface{}{},
		References:    map[string][]map[string]interface{}{},
	}
	requests, err := r.ListStudentRequests(ctx, profile.ID)
	if err != nil {
		return workspace, err
	}
	workspace.Requests = requests

	if profile.StudentID != nil && *profile.StudentID != "" {
		student, err := r.GetRecordByKey(ctx, "students", map[string]interface{}{"id": *profile.StudentID})
		if err != nil && err != pgx.ErrNoRows {
			return workspace, err
		}
		workspace.Student = student

		for _, table := range []string{"achievement_members", "student_tags", "project_members", "hof_competition_members", "hof_coordinators", "hof_internships"} {
			var records []map[string]interface{}
			var err error
			
			if table == "achievement_members" {
				// Join with achievements to get titles
				rows, queryErr := r.db.Pool().Query(ctx, `
					SELECT am.*, a.title as achievement_title, a.category as achievement_category
					FROM achievement_members am
					JOIN achievements a ON am.achievement_id = a.id
					WHERE am.student_id = $1
				`, *profile.StudentID)
				if queryErr != nil {
					err = queryErr
				} else {
					records, err = scanRecords(rows)
				}
			} else if table == "project_members" {
				// Join with projects to get titles
				rows, queryErr := r.db.Pool().Query(ctx, `
					SELECT pm.*, p.title as project_title
					FROM project_members pm
					JOIN projects p ON pm.project_id = p.id
					WHERE pm.student_id = $1
				`, *profile.StudentID)
				if queryErr != nil {
					err = queryErr
				} else {
					records, err = scanRecords(rows)
				}
			} else {
				records, err = r.listByStudentID(ctx, table, *profile.StudentID)
			}

			if err != nil {
				return workspace, err
			}
			workspace.LinkedRecords[table] = records
		}
	}

	for _, table := range []string{"batches", "tags", "projects", "hof_competitions", "achievements", "project_categories"} {
		records, err := r.listReferenceTable(ctx, table)
		if err != nil {
			return workspace, err
		}
		workspace.References[table] = records
	}

	return workspace, nil
}

func (r *StudentAccessRepository) CreateStudentRequest(ctx context.Context, profile model.UserProfile, req model.StudentChangeRequest) (model.StudentChangeRequest, error) {
	if !IsStudentOwnedTable(req.TargetTable) {
		return model.StudentChangeRequest{}, fmt.Errorf("students cannot request changes for %s", req.TargetTable)
	}
	if req.RequestType == "" {
		req.RequestType = "update"
	}
	if req.ProposedData == nil {
		req.ProposedData = map[string]interface{}{}
	}

	req.RequesterUserID = profile.ID
	req.Status = "pending"

	if req.TargetTable == "students" {
		if profile.StudentID == nil || *profile.StudentID == "" {
			if req.RequestType != "create" {
				return model.StudentChangeRequest{}, fmt.Errorf("student profile is not linked yet")
			}
		} else {
			req.StudentID = profile.StudentID
			req.TargetKey = map[string]interface{}{"id": *profile.StudentID}
			req.RequestType = "update"
			current, err := r.GetRecordByKey(ctx, "students", req.TargetKey)
			if err != nil {
				return model.StudentChangeRequest{}, err
			}
			req.CurrentData = current
		}
		return r.CreateChangeRequest(ctx, req)
	}

	if profile.StudentID == nil || *profile.StudentID == "" {
		return model.StudentChangeRequest{}, fmt.Errorf("student profile is not linked yet")
	}
	req.StudentID = profile.StudentID
	req.ProposedData["student_id"] = *profile.StudentID

	switch req.RequestType {
	case "create", "link":
		return r.CreateChangeRequest(ctx, req)
	case "update", "delete", "unlink":
		if req.TargetKey == nil {
			return model.StudentChangeRequest{}, fmt.Errorf("target_key is required")
		}
		current, err := r.GetRecordByKey(ctx, req.TargetTable, req.TargetKey)
		if err != nil {
			return model.StudentChangeRequest{}, err
		}
		// For tables that use junction tables for ownership, verify via the junction table
		if req.TargetTable == "projects" {
			isMember, memberErr := r.isProjectMember(ctx, *profile.StudentID, fmt.Sprintf("%v", current["id"]))
			if memberErr != nil {
				return model.StudentChangeRequest{}, memberErr
			}
			if !isMember {
				return model.StudentChangeRequest{}, fmt.Errorf("students can only change their own records")
			}
		} else if req.TargetTable == "achievements" {
			isMember, memberErr := r.isAchievementMember(ctx, *profile.StudentID, fmt.Sprintf("%v", current["id"]))
			if memberErr != nil {
				return model.StudentChangeRequest{}, memberErr
			}
			if !isMember {
				return model.StudentChangeRequest{}, fmt.Errorf("students can only change their own records")
			}
		} else if fmt.Sprintf("%v", current["student_id"]) != *profile.StudentID {
			return model.StudentChangeRequest{}, fmt.Errorf("students can only change their own records")
		}
		req.CurrentData = current
	default:
		return model.StudentChangeRequest{}, fmt.Errorf("unsupported request type: %s", req.RequestType)
	}

	return r.CreateChangeRequest(ctx, req)
}

func (r *StudentAccessRepository) CreateChangeRequest(ctx context.Context, req model.StudentChangeRequest) (model.StudentChangeRequest, error) {
	targetKeyJSON, err := marshalJSON(req.TargetKey)
	if err != nil {
		return model.StudentChangeRequest{}, err
	}
	currentJSON, err := marshalJSON(req.CurrentData)
	if err != nil {
		return model.StudentChangeRequest{}, err
	}
	proposedJSON, err := marshalJSON(req.ProposedData)
	if err != nil {
		return model.StudentChangeRequest{}, err
	}

	rows, err := r.db.Pool().Query(ctx, `
		INSERT INTO public.student_change_requests (
			requester_user_id, student_id, request_type, target_table, target_key, current_data, proposed_data
		)
		VALUES ($1, $2, $3, $4, $5::jsonb, $6::jsonb, $7::jsonb)
		RETURNING id::text, requester_user_id::text, student_id::text, request_type, target_table,
		          COALESCE(target_key, '{}'::jsonb)::text, COALESCE(current_data, '{}'::jsonb)::text,
		          proposed_data::text, status, COALESCE(admin_note, ''), reviewed_by::text,
		          reviewed_at, created_at, updated_at
	`, req.RequesterUserID, req.StudentID, req.RequestType, req.TargetTable, targetKeyJSON, currentJSON, proposedJSON)
	if err != nil {
		return model.StudentChangeRequest{}, err
	}
	defer rows.Close()
	return scanChangeRequest(rows)
}

func (r *StudentAccessRepository) ListStudentRequests(ctx context.Context, userID string) ([]model.StudentChangeRequest, error) {
	return r.listRequests(ctx, `
		SELECT id::text, requester_user_id::text, student_id::text, request_type, target_table,
		       COALESCE(target_key, '{}'::jsonb)::text, COALESCE(current_data, '{}'::jsonb)::text,
		       proposed_data::text, status, COALESCE(admin_note, ''), reviewed_by::text,
		       reviewed_at, created_at, updated_at
		FROM public.student_change_requests
		WHERE requester_user_id = $1
		ORDER BY created_at DESC
	`, userID)
}

func (r *StudentAccessRepository) ListAdminRequests(ctx context.Context, status string) ([]model.StudentChangeRequest, error) {
	baseQuery := `
		SELECT cr.id::text, cr.requester_user_id::text, cr.student_id::text, cr.request_type, cr.target_table,
		       COALESCE(cr.target_key, '{}'::jsonb)::text, COALESCE(cr.current_data, '{}'::jsonb)::text,
		       cr.proposed_data::text, cr.status, COALESCE(cr.admin_note, ''), cr.reviewed_by::text,
		       cr.reviewed_at, cr.created_at, cr.updated_at,
		       COALESCE(s.name, '') AS requester_name,
		       COALESCE(s.roll_no, '') AS requester_roll_no
		FROM public.student_change_requests cr
		LEFT JOIN public.profiles p ON p.id = cr.requester_user_id
		LEFT JOIN public.students s ON s.id = p.student_id
	`

	var rows pgx.Rows
	var err error
	if status == "" || status == "all" {
		rows, err = r.db.Pool().Query(ctx, baseQuery+"ORDER BY cr.created_at DESC")
	} else {
		rows, err = r.db.Pool().Query(ctx, baseQuery+"WHERE cr.status = $1 ORDER BY cr.created_at DESC", status)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	requests := make([]model.StudentChangeRequest, 0)
	for rows.Next() {
		var req model.StudentChangeRequest
		var targetKeyJSON, currentDataJSON, proposedDataJSON string
		var reviewedAt *time.Time
		var createdAt, updatedAt time.Time
		if err := rows.Scan(
			&req.ID,
			&req.RequesterUserID,
			&req.StudentID,
			&req.RequestType,
			&req.TargetTable,
			&targetKeyJSON,
			&currentDataJSON,
			&proposedDataJSON,
			&req.Status,
			&req.AdminNote,
			&req.ReviewedBy,
			&reviewedAt,
			&createdAt,
			&updatedAt,
			&req.RequesterName,
			&req.RequesterRollNo,
		); err != nil {
			return nil, err
		}
		req.TargetKey = parseJSONMap(targetKeyJSON)
		req.CurrentData = parseJSONMap(currentDataJSON)
		req.ProposedData = parseJSONMap(proposedDataJSON)
		if reviewedAt != nil {
			value := reviewedAt.Format(time.RFC3339)
			req.ReviewedAt = &value
		}
		req.CreatedAt = createdAt.Format(time.RFC3339)
		req.UpdatedAt = updatedAt.Format(time.RFC3339)
		requests = append(requests, req)
	}
	return requests, rows.Err()
}

func (r *StudentAccessRepository) listRequests(ctx context.Context, query string, args ...interface{}) ([]model.StudentChangeRequest, error) {
	rows, err := r.db.Pool().Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	requests := make([]model.StudentChangeRequest, 0)
	for rows.Next() {
		req, err := scanChangeRequestFromRows(rows)
		if err != nil {
			return nil, err
		}
		requests = append(requests, req)
	}
	return requests, rows.Err()
}

func (r *StudentAccessRepository) ApproveRequest(ctx context.Context, requestID, adminID, note string) (model.StudentChangeRequest, error) {
	tx, err := r.db.Pool().Begin(ctx)
	if err != nil {
		return model.StudentChangeRequest{}, err
	}
	defer tx.Rollback(ctx)

	req, err := r.getRequestForUpdate(ctx, tx, requestID)
	if err != nil {
		return model.StudentChangeRequest{}, err
	}
	if req.Status != "pending" {
		return model.StudentChangeRequest{}, fmt.Errorf("request is already %s", req.Status)
	}
	if err := r.applyRequest(ctx, tx, req); err != nil {
		return model.StudentChangeRequest{}, err
	}

	rows, err := tx.Query(ctx, `
		UPDATE public.student_change_requests
		SET status = 'approved',
		    admin_note = NULLIF($2, ''),
		    reviewed_by = $3,
		    reviewed_at = now(),
		    updated_at = now()
		WHERE id = $1
		RETURNING id::text, requester_user_id::text, student_id::text, request_type, target_table,
		          COALESCE(target_key, '{}'::jsonb)::text, COALESCE(current_data, '{}'::jsonb)::text,
		          proposed_data::text, status, COALESCE(admin_note, ''), reviewed_by::text,
		          reviewed_at, created_at, updated_at
	`, requestID, note, adminID)
	if err != nil {
		return model.StudentChangeRequest{}, err
	}

	updated, err := scanChangeRequest(rows)
	rows.Close()
	if err != nil {
		return model.StudentChangeRequest{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return model.StudentChangeRequest{}, err
	}
	return updated, nil
}

func (r *StudentAccessRepository) RejectRequest(ctx context.Context, requestID, adminID, note string) (model.StudentChangeRequest, error) {
	rows, err := r.db.Pool().Query(ctx, `
		UPDATE public.student_change_requests
		SET status = 'rejected',
		    admin_note = NULLIF($2, ''),
		    reviewed_by = $3,
		    reviewed_at = now(),
		    updated_at = now()
		WHERE id = $1
		  AND status = 'pending'
		RETURNING id::text, requester_user_id::text, student_id::text, request_type, target_table,
		          COALESCE(target_key, '{}'::jsonb)::text, COALESCE(current_data, '{}'::jsonb)::text,
		          proposed_data::text, status, COALESCE(admin_note, ''), reviewed_by::text,
		          reviewed_at, created_at, updated_at
	`, requestID, note, adminID)
	if err != nil {
		return model.StudentChangeRequest{}, err
	}
	defer rows.Close()
	return scanChangeRequest(rows)
}

func (r *StudentAccessRepository) getRequestForUpdate(ctx context.Context, tx pgx.Tx, requestID string) (model.StudentChangeRequest, error) {
	rows, err := tx.Query(ctx, `
		SELECT id::text, requester_user_id::text, student_id::text, request_type, target_table,
		       COALESCE(target_key, '{}'::jsonb)::text, COALESCE(current_data, '{}'::jsonb)::text,
		       proposed_data::text, status, COALESCE(admin_note, ''), reviewed_by::text,
		       reviewed_at, created_at, updated_at
		FROM public.student_change_requests
		WHERE id = $1
		FOR UPDATE
	`, requestID)
	if err != nil {
		return model.StudentChangeRequest{}, err
	}
	defer rows.Close()
	return scanChangeRequest(rows)
}

func (r *StudentAccessRepository) applyRequest(ctx context.Context, tx pgx.Tx, req model.StudentChangeRequest) error {
	switch req.RequestType {
	case "claim_student":
		studentID, ok := req.TargetKey["id"]
		if !ok || fmt.Sprintf("%v", studentID) == "" {
			return fmt.Errorf("claim request is missing student id")
		}
		claimed, err := r.studentClaimedTx(ctx, tx, fmt.Sprintf("%v", studentID), req.RequesterUserID)
		if err != nil {
			return err
		}
		if claimed {
			return fmt.Errorf("student is already claimed")
		}
		_, err = tx.Exec(ctx, `UPDATE public.profiles SET student_id = $1 WHERE id = $2`, studentID, req.RequesterUserID)
		return err
	case "create", "link":
		if req.TargetTable == "achievement_members" {
			studentRaw, hasStudent := req.ProposedData["student_id"]
			achievementRaw, hasAchievement := req.ProposedData["achievement_id"]
			if !hasStudent || !hasAchievement || studentRaw == nil || achievementRaw == nil {
				return fmt.Errorf("student_id and achievement_id are required")
			}
			studentID := strings.TrimSpace(fmt.Sprintf("%v", studentRaw))
			achievementID := strings.TrimSpace(fmt.Sprintf("%v", achievementRaw))
			if studentID == "" || achievementID == "" {
				return fmt.Errorf("student_id and achievement_id are required")
			}
			_, err := tx.Exec(ctx, `
				INSERT INTO public.achievement_members (student_id, achievement_id)
				VALUES ($1, $2)
				ON CONFLICT (student_id, achievement_id) DO NOTHING
			`, studentID, achievementID)
			return err
		}
		if req.TargetTable == "project_members" {
			studentRaw, hasStudent := req.ProposedData["student_id"]
			projectRaw, hasProject := req.ProposedData["project_id"]
			if !hasStudent || !hasProject || studentRaw == nil || projectRaw == nil {
				return fmt.Errorf("student_id and project_id are required")
			}
			studentID := strings.TrimSpace(fmt.Sprintf("%v", studentRaw))
			projectID := strings.TrimSpace(fmt.Sprintf("%v", projectRaw))
			role := ""
			if roleRaw, ok := req.ProposedData["role"]; ok && roleRaw != nil {
				role = strings.TrimSpace(fmt.Sprintf("%v", roleRaw))
			}
			if studentID == "" || projectID == "" {
				return fmt.Errorf("student_id and project_id are required")
			}
			_, err := tx.Exec(ctx, `
				INSERT INTO public.project_members (student_id, project_id, role)
				VALUES ($1, $2, NULLIF($3, ''))
				ON CONFLICT (student_id, project_id) DO UPDATE SET role = EXCLUDED.role
			`, studentID, projectID, role)
			return err
		}
		// Handle junction table migration for student-owned records
		if req.TargetTable == "achievements" {
			// Extract student_id if present (for junction table migration)
			var studentID string
			proposedData := make(map[string]interface{})
			for k, v := range req.ProposedData {
				if k == "student_id" {
					studentID = fmt.Sprintf("%v", v)
				} else {
					proposedData[k] = v
				}
			}
			inserted, err := insertGeneric(ctx, tx, req.TargetTable, proposedData)
			if err != nil {
				return err
			}
			// If student_id was provided, create junction table entry
			if studentID != "" {
				achievementID := fmt.Sprintf("%v", inserted["id"])
				_, err = tx.Exec(ctx, `INSERT INTO public.achievement_members (student_id, achievement_id) VALUES ($1, $2) ON CONFLICT (student_id, achievement_id) DO NOTHING`, studentID, achievementID)
				if err != nil {
					return fmt.Errorf("failed to create achievement_member: %w", err)
				}
			}
			return nil
		}
		// Handle junction table migration for student-owned projects
		if req.TargetTable == "projects" {
			var studentID string
			var role string
			proposedData := make(map[string]interface{})
			for k, v := range req.ProposedData {
				if k == "student_id" {
					studentID = fmt.Sprintf("%v", v)
				} else if k == "member_role" {
					role = fmt.Sprintf("%v", v)
				} else {
					proposedData[k] = v
				}
			}
			inserted, err := insertGeneric(ctx, tx, req.TargetTable, proposedData)
			if err != nil {
				return err
			}
			// If student_id was provided, create junction table entry
			if studentID != "" {
				projectID := fmt.Sprintf("%v", inserted["id"])
				_, err = tx.Exec(ctx, `INSERT INTO public.project_members (student_id, project_id, role) VALUES ($1, $2, NULLIF($3, '')) ON CONFLICT (student_id, project_id) DO NOTHING`, studentID, projectID, role)
				if err != nil {
					return fmt.Errorf("failed to create project_member: %w", err)
				}
			}
			return nil
		}
		inserted, err := insertGeneric(ctx, tx, req.TargetTable, req.ProposedData)
		if err != nil {
			return err
		}
		if req.TargetTable == "students" {
			studentID := fmt.Sprintf("%v", inserted["id"])
			_, err = tx.Exec(ctx, `UPDATE public.profiles SET student_id = $1 WHERE id = $2`, studentID, req.RequesterUserID)
			return err
		}
		return nil
	case "update":
		return updateGeneric(ctx, tx, req.TargetTable, req.TargetKey, req.ProposedData)
	case "delete", "unlink":
		return deleteGeneric(ctx, tx, req.TargetTable, req.TargetKey)
	default:
		return fmt.Errorf("unsupported request type: %s", req.RequestType)
	}
}

func (r *StudentAccessRepository) studentClaimedTx(ctx context.Context, tx pgx.Tx, studentID, requesterUserID string) (bool, error) {
	var count int
	err := tx.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM public.profiles
		WHERE student_id = $1
		  AND id::text <> $2
	`, studentID, requesterUserID).Scan(&count)
	return count > 0, err
}

func (r *StudentAccessRepository) GetRecordByKey(ctx context.Context, table string, key map[string]interface{}) (map[string]interface{}, error) {
	if !isValidIdentifier(table) {
		return nil, fmt.Errorf("invalid table")
	}
	where, args, err := buildGenericWhere(key, 1)
	if err != nil {
		return nil, err
	}
	rows, err := r.db.Pool().Query(ctx, fmt.Sprintf("SELECT * FROM %s WHERE %s LIMIT 1", quoteIdentifier(table), where), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	record, err := scanOneRecord(rows)
	if err != nil {
		return nil, err
	}
	if record == nil {
		return nil, pgx.ErrNoRows
	}
	return record, nil
}

func (r *StudentAccessRepository) listByStudentID(ctx context.Context, table, studentID string) ([]map[string]interface{}, error) {
	rows, err := r.db.Pool().Query(ctx, fmt.Sprintf("SELECT * FROM %s WHERE student_id = $1", quoteIdentifier(table)), studentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanRecords(rows)
}

func (r *StudentAccessRepository) listReferenceTable(ctx context.Context, table string) ([]map[string]interface{}, error) {
	rows, err := r.db.Pool().Query(ctx, fmt.Sprintf("SELECT * FROM %s LIMIT 500", quoteIdentifier(table)))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanRecords(rows)
}

func insertGeneric(ctx context.Context, tx pgx.Tx, table string, data map[string]interface{}) (map[string]interface{}, error) {
	if !isValidIdentifier(table) {
		return nil, fmt.Errorf("invalid table")
	}
	columns, placeholders, values := buildInsertParts(data)
	if len(columns) == 0 {
		return nil, fmt.Errorf("no data provided")
	}
	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s) RETURNING *", quoteIdentifier(table), strings.Join(columns, ", "), strings.Join(placeholders, ", "))
	rows, err := tx.Query(ctx, query, values...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanOneRecord(rows)
}

func updateGeneric(ctx context.Context, tx pgx.Tx, table string, key, data map[string]interface{}) error {
	if !isValidIdentifier(table) {
		return fmt.Errorf("invalid table")
	}
	setClauses, values := buildUpdateParts(data, key)
	if len(setClauses) == 0 {
		return fmt.Errorf("no updateable data provided")
	}
	where, whereArgs, err := buildGenericWhere(key, len(values)+1)
	if err != nil {
		return err
	}
	values = append(values, whereArgs...)
	query := fmt.Sprintf("UPDATE %s SET %s WHERE %s", quoteIdentifier(table), strings.Join(setClauses, ", "), where)
	result, err := tx.Exec(ctx, query, values...)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("record not found")
	}
	return nil
}

func deleteGeneric(ctx context.Context, tx pgx.Tx, table string, key map[string]interface{}) error {
	if !isValidIdentifier(table) {
		return fmt.Errorf("invalid table")
	}
	where, values, err := buildGenericWhere(key, 1)
	if err != nil {
		return err
	}
	result, err := tx.Exec(ctx, fmt.Sprintf("DELETE FROM %s WHERE %s", quoteIdentifier(table), where), values...)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("record not found")
	}
	return nil
}

func buildInsertParts(data map[string]interface{}) ([]string, []string, []interface{}) {
	keys := sortedValidKeys(data)
	columns := make([]string, 0, len(keys))
	placeholders := make([]string, 0, len(keys))
	values := make([]interface{}, 0, len(keys))
	for idx, key := range keys {
		columns = append(columns, quoteIdentifier(key))
		placeholders = append(placeholders, fmt.Sprintf("$%d", idx+1))
		values = append(values, data[key])
	}
	return columns, placeholders, values
}

func buildUpdateParts(data, key map[string]interface{}) ([]string, []interface{}) {
	keySet := map[string]bool{}
	for k := range key {
		keySet[k] = true
	}
	keys := sortedValidKeys(data)
	setClauses := make([]string, 0, len(keys))
	values := make([]interface{}, 0, len(keys))
	for _, col := range keys {
		if keySet[col] || col == "id" || col == "student_id" || col == "created_at" {
			continue
		}
		values = append(values, data[col])
		setClauses = append(setClauses, fmt.Sprintf("%s = $%d", quoteIdentifier(col), len(values)))
	}
	return setClauses, values
}

func buildGenericWhere(key map[string]interface{}, start int) (string, []interface{}, error) {
	keys := sortedValidKeys(key)
	if len(keys) == 0 {
		return "", nil, fmt.Errorf("target key is required")
	}
	conditions := make([]string, 0, len(keys))
	values := make([]interface{}, 0, len(keys))
	for i, k := range keys {
		conditions = append(conditions, fmt.Sprintf("%s::text = $%d", quoteIdentifier(k), start+i))
		values = append(values, fmt.Sprintf("%v", key[k]))
	}
	return strings.Join(conditions, " AND "), values, nil
}

func sortedValidKeys(data map[string]interface{}) []string {
	keys := make([]string, 0, len(data))
	for key := range data {
		if isValidIdentifier(key) {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	return keys
}

func marshalJSON(value map[string]interface{}) (string, error) {
	if value == nil {
		return "null", nil
	}
	b, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func scanChangeRequest(rows pgx.Rows) (model.StudentChangeRequest, error) {
	if !rows.Next() {
		if rows.Err() != nil {
			return model.StudentChangeRequest{}, rows.Err()
		}
		return model.StudentChangeRequest{}, pgx.ErrNoRows
	}
	return scanChangeRequestFromRows(rows)
}

func scanChangeRequestFromRows(rows pgx.Rows) (model.StudentChangeRequest, error) {
	var req model.StudentChangeRequest
	var targetKeyJSON, currentDataJSON, proposedDataJSON string
	var reviewedAt *time.Time
	var createdAt, updatedAt time.Time
	if err := rows.Scan(
		&req.ID,
		&req.RequesterUserID,
		&req.StudentID,
		&req.RequestType,
		&req.TargetTable,
		&targetKeyJSON,
		&currentDataJSON,
		&proposedDataJSON,
		&req.Status,
		&req.AdminNote,
		&req.ReviewedBy,
		&reviewedAt,
		&createdAt,
		&updatedAt,
	); err != nil {
		return req, err
	}
	req.TargetKey = parseJSONMap(targetKeyJSON)
	req.CurrentData = parseJSONMap(currentDataJSON)
	req.ProposedData = parseJSONMap(proposedDataJSON)
	if reviewedAt != nil {
		value := reviewedAt.Format(time.RFC3339)
		req.ReviewedAt = &value
	}
	req.CreatedAt = createdAt.Format(time.RFC3339)
	req.UpdatedAt = updatedAt.Format(time.RFC3339)
	return req, nil
}

func (r *StudentAccessRepository) getRequesterStudentID(ctx context.Context, requesterID string) (string, error) {
	var studentID *string
	err := r.db.Pool().QueryRow(ctx, `SELECT student_id FROM public.profiles WHERE id = $1`, requesterID).Scan(&studentID)
	if err != nil {
		return "", fmt.Errorf("failed to get requester student: %w", err)
	}
	if studentID == nil || *studentID == "" {
		return "", fmt.Errorf("you must have a student profile to perform this action")
	}
	return *studentID, nil
}

func (r *StudentAccessRepository) recordExists(ctx context.Context, table string, id string) (bool, error) {
	if !isValidIdentifier(table) {
		return false, fmt.Errorf("invalid table")
	}
	var exists bool
	query := fmt.Sprintf("SELECT EXISTS(SELECT 1 FROM %s WHERE id = $1)", quoteIdentifier(table))
	err := r.db.Pool().QueryRow(ctx, query, id).Scan(&exists)
	return exists, err
}

func (r *StudentAccessRepository) isAchievementMember(ctx context.Context, studentID, achievementID string) (bool, error) {
	var exists bool
	err := r.db.Pool().QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM public.achievement_members
			WHERE achievement_id = $1 AND student_id = $2
		)
	`, achievementID, studentID).Scan(&exists)
	return exists, err
}

func (r *StudentAccessRepository) isProjectMember(ctx context.Context, studentID, projectID string) (bool, error) {
	var exists bool
	err := r.db.Pool().QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM public.project_members
			WHERE project_id = $1 AND student_id = $2
		)
	`, projectID, studentID).Scan(&exists)
	return exists, err
}

func (r *StudentAccessRepository) hasPendingContributorRequest(ctx context.Context, targetTable, recordID, studentID string) (bool, error) {
	var exists bool
	if targetTable == "achievement_members" {
		err := r.db.Pool().QueryRow(ctx, `
			SELECT EXISTS(
				SELECT 1 FROM public.student_change_requests
				WHERE status = 'pending'
				  AND request_type = 'link'
				  AND target_table = 'achievement_members'
				  AND proposed_data->>'student_id' = $1
				  AND proposed_data->>'achievement_id' = $2
			)
		`, studentID, recordID).Scan(&exists)
		return exists, err
	}
	if targetTable == "project_members" {
		err := r.db.Pool().QueryRow(ctx, `
			SELECT EXISTS(
				SELECT 1 FROM public.student_change_requests
				WHERE status = 'pending'
				  AND request_type = 'link'
				  AND target_table = 'project_members'
				  AND proposed_data->>'student_id' = $1
				  AND proposed_data->>'project_id' = $2
			)
		`, studentID, recordID).Scan(&exists)
		return exists, err
	}
	return false, fmt.Errorf("unsupported target table")
}

func (r *StudentAccessRepository) getStudentDisplay(ctx context.Context, studentID string) (string, string, error) {
	var name, rollNo string
	err := r.db.Pool().QueryRow(ctx, `
		SELECT COALESCE(name, ''), COALESCE(roll_no, '')
		FROM public.students
		WHERE id = $1
	`, studentID).Scan(&name, &rollNo)
	if err != nil {
		return "", "", err
	}
	return name, rollNo, nil
}

func (r *StudentAccessRepository) getAchievementTitle(ctx context.Context, achievementID string) (string, error) {
	var title string
	err := r.db.Pool().QueryRow(ctx, `
		SELECT COALESCE(title, '')
		FROM public.achievements
		WHERE id = $1
	`, achievementID).Scan(&title)
	return title, err
}

func (r *StudentAccessRepository) getProjectTitle(ctx context.Context, projectID string) (string, error) {
	var title string
	err := r.db.Pool().QueryRow(ctx, `
		SELECT COALESCE(title, '')
		FROM public.projects
		WHERE id = $1
	`, projectID).Scan(&title)
	return title, err
}

// AddAchievementContributor requests a contributor addition for admin approval.
func (r *StudentAccessRepository) AddAchievementContributor(ctx context.Context, requesterID string, achievementID, studentID string) (model.StudentChangeRequest, error) {
	requesterStudentID, err := r.getRequesterStudentID(ctx, requesterID)
	if err != nil {
		return model.StudentChangeRequest{}, err
	}
	if studentID == "" {
		studentID = requesterStudentID
	}
	if achievementID == "" || studentID == "" {
		return model.StudentChangeRequest{}, fmt.Errorf("achievement_id and student_id are required")
	}

	exists, err := r.recordExists(ctx, "achievements", achievementID)
	if err != nil {
		return model.StudentChangeRequest{}, err
	}
	if !exists {
		return model.StudentChangeRequest{}, fmt.Errorf("achievement not found")
	}

	exists, err = r.recordExists(ctx, "students", studentID)
	if err != nil {
		return model.StudentChangeRequest{}, err
	}
	if !exists {
		return model.StudentChangeRequest{}, fmt.Errorf("student not found")
	}

	isMember, err := r.isAchievementMember(ctx, studentID, achievementID)
	if err != nil {
		return model.StudentChangeRequest{}, err
	}
	if isMember {
		return model.StudentChangeRequest{}, fmt.Errorf("student is already a contributor")
	}

	requesterIsMember, err := r.isAchievementMember(ctx, requesterStudentID, achievementID)
	if err != nil {
		return model.StudentChangeRequest{}, err
	}
	if requesterStudentID != studentID && !requesterIsMember {
		return model.StudentChangeRequest{}, fmt.Errorf("you must be a contributor to add others")
	}

	pending, err := r.hasPendingContributorRequest(ctx, "achievement_members", achievementID, studentID)
	if err != nil {
		return model.StudentChangeRequest{}, err
	}
	if pending {
		return model.StudentChangeRequest{}, fmt.Errorf("a pending request already exists")
	}

	studentName, studentRollNo, err := r.getStudentDisplay(ctx, studentID)
	if err != nil {
		return model.StudentChangeRequest{}, err
	}
	achievementTitle, err := r.getAchievementTitle(ctx, achievementID)
	if err != nil {
		return model.StudentChangeRequest{}, err
	}

	request := model.StudentChangeRequest{
		RequesterUserID: requesterID,
		StudentID:       &studentID,
		RequestType:     "link",
		TargetTable:     "achievement_members",
		TargetKey: map[string]interface{}{
			"achievement_id": achievementID,
			"student_id":     studentID,
		},
		ProposedData: map[string]interface{}{
			"achievement_id":    achievementID,
			"achievement_title": achievementTitle,
			"student_id":        studentID,
			"student_name":      studentName,
			"student_roll_no":   studentRollNo,
		},
		Status: "pending",
	}

	return r.CreateChangeRequest(ctx, request)
}

// AddProjectContributor requests a contributor addition for admin approval.
func (r *StudentAccessRepository) AddProjectContributor(ctx context.Context, requesterID string, projectID, studentID string, role string) (model.StudentChangeRequest, error) {
	requesterStudentID, err := r.getRequesterStudentID(ctx, requesterID)
	if err != nil {
		return model.StudentChangeRequest{}, err
	}
	if studentID == "" {
		studentID = requesterStudentID
	}
	if projectID == "" || studentID == "" {
		return model.StudentChangeRequest{}, fmt.Errorf("project_id and student_id are required")
	}

	exists, err := r.recordExists(ctx, "projects", projectID)
	if err != nil {
		return model.StudentChangeRequest{}, err
	}
	if !exists {
		return model.StudentChangeRequest{}, fmt.Errorf("project not found")
	}

	exists, err = r.recordExists(ctx, "students", studentID)
	if err != nil {
		return model.StudentChangeRequest{}, err
	}
	if !exists {
		return model.StudentChangeRequest{}, fmt.Errorf("student not found")
	}

	isMember, err := r.isProjectMember(ctx, studentID, projectID)
	if err != nil {
		return model.StudentChangeRequest{}, err
	}
	if isMember {
		return model.StudentChangeRequest{}, fmt.Errorf("student is already a contributor")
	}

	requesterIsMember, err := r.isProjectMember(ctx, requesterStudentID, projectID)
	if err != nil {
		return model.StudentChangeRequest{}, err
	}
	if requesterStudentID != studentID && !requesterIsMember {
		return model.StudentChangeRequest{}, fmt.Errorf("you must be a contributor to add others")
	}

	pending, err := r.hasPendingContributorRequest(ctx, "project_members", projectID, studentID)
	if err != nil {
		return model.StudentChangeRequest{}, err
	}
	if pending {
		return model.StudentChangeRequest{}, fmt.Errorf("a pending request already exists")
	}

	studentName, studentRollNo, err := r.getStudentDisplay(ctx, studentID)
	if err != nil {
		return model.StudentChangeRequest{}, err
	}
	projectTitle, err := r.getProjectTitle(ctx, projectID)
	if err != nil {
		return model.StudentChangeRequest{}, err
	}

	request := model.StudentChangeRequest{
		RequesterUserID: requesterID,
		StudentID:       &studentID,
		RequestType:     "link",
		TargetTable:     "project_members",
		TargetKey: map[string]interface{}{
			"project_id": projectID,
			"student_id": studentID,
		},
		ProposedData: map[string]interface{}{
			"project_id":    projectID,
			"project_title": projectTitle,
			"student_id":    studentID,
			"student_name":  studentName,
			"student_roll_no": studentRollNo,
			"role":          role,
		},
		Status: "pending",
	}

	return r.CreateChangeRequest(ctx, request)
}

// RemoveAchievementContributor removes a student from an achievement
func (r *StudentAccessRepository) RemoveAchievementContributor(ctx context.Context, requesterID string, achievementID, studentID string) error {
	// Verify the requester owns this achievement
	var requesterStudentID *string
	err := r.db.Pool().QueryRow(ctx, `SELECT student_id FROM public.profiles WHERE id = $1`, requesterID).Scan(&requesterStudentID)
	if err != nil {
		return fmt.Errorf("failed to verify ownership: %w", err)
	}
	if requesterStudentID == nil || *requesterStudentID == "" {
		return fmt.Errorf("you must have a student profile to perform this action")
	}

	var count int
	err = r.db.Pool().QueryRow(ctx, `
		SELECT COUNT(*)
		FROM public.achievement_members am
		WHERE am.achievement_id = $1 AND am.student_id = $2
	`, achievementID, *requesterStudentID).Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to verify ownership: %w", err)
	}
	if count == 0 {
		return fmt.Errorf("you are not an owner of this achievement")
	}

	// Remove contributor
	result, err := r.db.Pool().Exec(ctx, `
		DELETE FROM public.achievement_members
		WHERE student_id = $1 AND achievement_id = $2
	`, studentID, achievementID)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("contributor not found")
	}
	return nil
}

// RemoveProjectContributor removes a student from a project
func (r *StudentAccessRepository) RemoveProjectContributor(ctx context.Context, requesterID string, projectID, studentID string) error {
	// Verify the requester owns this project
	var requesterStudentID *string
	err := r.db.Pool().QueryRow(ctx, `SELECT student_id FROM public.profiles WHERE id = $1`, requesterID).Scan(&requesterStudentID)
	if err != nil {
		return fmt.Errorf("failed to verify ownership: %w", err)
	}
	if requesterStudentID == nil || *requesterStudentID == "" {
		return fmt.Errorf("you must have a student profile to perform this action")
	}

	var count int
	err = r.db.Pool().QueryRow(ctx, `
		SELECT COUNT(*)
		FROM public.project_members pm
		WHERE pm.project_id = $1 AND pm.student_id = $2
	`, projectID, *requesterStudentID).Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to verify ownership: %w", err)
	}
	if count == 0 {
		return fmt.Errorf("you are not an owner of this project")
	}

	// Remove contributor
	result, err := r.db.Pool().Exec(ctx, `
		DELETE FROM public.project_members
		WHERE student_id = $1 AND project_id = $2
	`, studentID, projectID)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("contributor not found")
	}
	return nil
}

// ListStudentsSearchable returns students matching a search term (for adding contributors)
func (r *StudentAccessRepository) ListStudentsSearchable(ctx context.Context, search string, limit int) ([]map[string]interface{}, error) {
	if limit == 0 || limit > 50 {
		limit = 20
	}
	query := `
		SELECT id, name, roll_no, batch_id, image_url
		FROM public.students
		WHERE name ILIKE $1 OR roll_no ILIKE $1
		ORDER BY name
		LIMIT $2
	`
	rows, err := r.db.Pool().Query(ctx, query, "%"+search+"%", limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanRecords(rows)
}

// GetStudentContributors returns all students who can be added as contributors
func (r *StudentAccessRepository) GetStudentContributors(ctx context.Context, requesterID, recordType, recordID string) ([]map[string]interface{}, error) {
	// First verify requester owns this record
	var requesterStudentID *string
	err := r.db.Pool().QueryRow(ctx, `SELECT student_id FROM public.profiles WHERE id = $1`, requesterID).Scan(&requesterStudentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get requester student: %w", err)
	}
	if requesterStudentID == nil || *requesterStudentID == "" {
		return nil, fmt.Errorf("you don't have a student profile linked")
	}

	var owned bool
	if recordType == "achievement" {
		err = r.db.Pool().QueryRow(ctx, `
			SELECT EXISTS(SELECT 1 FROM public.achievement_members WHERE achievement_id = $1 AND student_id = $2)
		`, recordID, *requesterStudentID).Scan(&owned)
	} else if recordType == "project" {
		err = r.db.Pool().QueryRow(ctx, `
			SELECT EXISTS(SELECT 1 FROM public.project_members WHERE project_id = $1 AND student_id = $2)
		`, recordID, *requesterStudentID).Scan(&owned)
	} else {
		return nil, fmt.Errorf("invalid record type")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to verify ownership: %w", err)
	}
	if !owned {
		return nil, fmt.Errorf("you don't own this record")
	}

	// Get existing contributors
	var query string
	if recordType == "achievement" {
		// achievement_members doesn't have a role column
		query = `
			SELECT s.id, s.name, s.roll_no, s.image_url, NULL::text AS role
			FROM public.students s
			INNER JOIN public.achievement_members am ON s.id = am.student_id
			WHERE am.achievement_id = $1
			ORDER BY s.name
		`
	} else {
		// project_members has a role column
		query = `
			SELECT s.id, s.name, s.roll_no, s.image_url, pm.role
			FROM public.students s
			INNER JOIN public.project_members pm ON s.id = pm.student_id
			WHERE pm.project_id = $1
			ORDER BY s.name
		`
	}

	rows, err := r.db.Pool().Query(ctx, query, recordID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanRecords(rows)
}

func parseJSONMap(raw string) map[string]interface{} {
	if raw == "" || raw == "null" {
		return map[string]interface{}{}
	}
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &data); err != nil {
		return map[string]interface{}{}
	}
	return data
}

func scanOneRecord(rows pgx.Rows) (map[string]interface{}, error) {
	records, err := scanRecords(rows)
	if err != nil || len(records) == 0 {
		return nil, err
	}
	return records[0], nil
}

func scanRecords(rows pgx.Rows) ([]map[string]interface{}, error) {
	records := make([]map[string]interface{}, 0)
	for rows.Next() {
		values, err := rows.Values()
		if err != nil {
			return nil, err
		}
		fields := rows.FieldDescriptions()
		record := make(map[string]interface{}, len(fields))
		for i, fd := range fields {
			if i < len(values) {
				record[fd.Name] = convertValue(values[i])
			}
		}
		records = append(records, record)
	}
	return records, rows.Err()
}
