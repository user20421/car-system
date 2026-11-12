package store

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/car-system/car-system/backend-go/internal/domain"
)

const vehicleSelect = `SELECT v.vin,v.display_name,v.model,v.model_year,v.owner_user_id,
CASE WHEN ? THEN 'owner' WHEN v.owner_user_id=? THEN 'owner' ELSE COALESCE(vs.access_role,'') END,
COALESCE(v.firmware_version,''),v.created_at FROM vehicles v LEFT JOIN vehicle_shares vs ON vs.vin=v.vin AND vs.user_id=? `

func scanVehicle(scanner interface{ Scan(...any) error }) (domain.Vehicle, error) {
	var v domain.Vehicle
	err := scanner.Scan(&v.VIN, &v.DisplayName, &v.Model, &v.ModelYear, &v.OwnerUserID, &v.AccessRole, &v.FirmwareVersion, &v.CreatedAt)
	if err == sql.ErrNoRows {
		return v, domain.ErrNotFound
	}
	return v, err
}
func (s *Store) VehicleForUser(ctx context.Context, user domain.User, vin string) (domain.Vehicle, error) {
	v, err := scanVehicle(s.DB.QueryRowContext(ctx, vehicleSelect+"WHERE v.vin=? AND (? OR v.owner_user_id=? OR vs.user_id IS NOT NULL)", user.Admin(), user.ID, user.ID, vin, user.Admin(), user.ID))
	return v, err
}
func (s *Store) VehicleByVIN(ctx context.Context, vin string) (domain.Vehicle, error) {
	return scanVehicle(s.DB.QueryRowContext(ctx, vehicleSelect+"WHERE v.vin=?", true, "", "", vin))
}
func (s *Store) ListVehicles(ctx context.Context, user domain.User, page, size int) ([]domain.Vehicle, int, error) {
	var total int
	if err := s.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM vehicles v LEFT JOIN vehicle_shares vs ON vs.vin=v.vin AND vs.user_id=? WHERE ? OR v.owner_user_id=? OR vs.user_id IS NOT NULL`, user.ID, user.Admin(), user.ID).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := s.DB.QueryContext(ctx, vehicleSelect+"WHERE ? OR v.owner_user_id=? OR vs.user_id IS NOT NULL ORDER BY v.vin LIMIT ? OFFSET ?", user.Admin(), user.ID, user.ID, user.Admin(), user.ID, size, (page-1)*size)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items := []domain.Vehicle{}
	for rows.Next() {
		v, err := scanVehicle(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, v)
	}
	return items, total, rows.Err()
}

type DeviceConfig struct {
	VIN                  string `json:"vin"`
	ConfigVersion        int64  `json:"config_version"`
	TelemetryIntervalMS  int    `json:"telemetry_interval_ms"`
	ConfigPollIntervalMS int    `json:"config_poll_interval_ms"`
	MQTTTopicVersion     string `json:"mqtt_topic_version"`
	UpdatedAtMS          int64  `json:"updated_at_ms"`
}

func (s *Store) DeviceConfig(ctx context.Context, vin string) (DeviceConfig, error) {
	var c DeviceConfig
	c.MQTTTopicVersion = "v1"
	err := s.DB.QueryRowContext(ctx, "SELECT vin,config_version,telemetry_interval_ms,config_poll_interval_ms,updated_at_ms FROM device_configs WHERE vin=?", vin).Scan(&c.VIN, &c.ConfigVersion, &c.TelemetryIntervalMS, &c.ConfigPollIntervalMS, &c.UpdatedAtMS)
	if err == sql.ErrNoRows {
		err = domain.ErrNotFound
	}
	return c, err
}

type OTARelease struct {
	Version      string `json:"version"`
	URL          string `json:"url"`
	SHA256       string `json:"sha256"`
	SizeBytes    int64  `json:"size_bytes"`
	Mandatory    bool   `json:"mandatory"`
	ReleaseNotes string `json:"release_notes,omitempty"`
	ReleasedAtMS int64  `json:"released_at_ms"`
}

func (s *Store) LatestOTA(ctx context.Context, vin string) (*OTARelease, error) {
	var model string
	if err := s.DB.QueryRowContext(ctx, "SELECT model FROM vehicles WHERE vin=?", vin).Scan(&model); err != nil {
		return nil, err
	}
	var r OTARelease
	var notes sql.NullString
	err := s.DB.QueryRowContext(ctx, "SELECT version,url,sha256,size_bytes,mandatory,release_notes,released_at_ms FROM ota_releases WHERE model=? ORDER BY released_at_ms DESC LIMIT 1", model).Scan(&r.Version, &r.URL, &r.SHA256, &r.SizeBytes, &r.Mandatory, &notes, &r.ReleasedAtMS)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	r.ReleaseNotes = notes.String
	return &r, nil
}

func (s *Store) ListAlerts(ctx context.Context, user domain.User, vin, status string, page, size int) ([]domain.Alert, int, error) {
	where := ` FROM alerts a JOIN vehicles v ON v.vin=a.vin LEFT JOIN vehicle_shares vs ON vs.vin=a.vin AND vs.user_id=? WHERE (? OR v.owner_user_id=? OR vs.user_id IS NOT NULL)`
	args := []any{user.ID, user.Admin(), user.ID}
	if vin != "" {
		where += " AND a.vin=?"
		args = append(args, vin)
	}
	if status != "" {
		where += " AND a.status=?"
		args = append(args, status)
	}
	var total int
	if err := s.DB.QueryRowContext(ctx, "SELECT COUNT(*)"+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	query := `SELECT a.id,a.vin,a.code,a.title,a.severity,a.status,a.occurred_at_ms,a.acknowledged_by,a.acknowledged_at_ms,a.acknowledgement_note` + where + ` ORDER BY a.occurred_at_ms DESC LIMIT ? OFFSET ?`
	args = append(args, size, (page-1)*size)
	rows, err := s.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items := []domain.Alert{}
	for rows.Next() {
		a, err := scanAlert(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, a)
	}
	return items, total, rows.Err()
}
func scanAlert(scanner interface{ Scan(...any) error }) (domain.Alert, error) {
	var a domain.Alert
	var by, note sql.NullString
	var at sql.NullInt64
	err := scanner.Scan(&a.ID, &a.VIN, &a.Code, &a.Title, &a.Severity, &a.Status, &a.OccurredAtMS, &by, &at, &note)
	if err == sql.ErrNoRows {
		return a, domain.ErrNotFound
	}
	if err != nil {
		return a, err
	}
	if by.Valid {
		a.AcknowledgedBy = &by.String
	}
	if at.Valid {
		a.AcknowledgedAtMS = &at.Int64
	}
	if note.Valid {
		a.AcknowledgementNote = &note.String
	}
	return a, nil
}
func (s *Store) AcknowledgeAlert(ctx context.Context, user domain.User, id, note string, now int64) (domain.Alert, error) {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return domain.Alert{}, err
	}
	defer tx.Rollback()
	var vin, current string
	err = tx.QueryRowContext(ctx, `SELECT a.vin,a.status FROM alerts a JOIN vehicles v ON v.vin=a.vin LEFT JOIN vehicle_shares vs ON vs.vin=a.vin AND vs.user_id=? WHERE a.id=? AND (? OR v.owner_user_id=? OR (vs.user_id IS NOT NULL AND vs.access_role='driver')) FOR UPDATE`, user.ID, id, user.Admin(), user.ID).Scan(&vin, &current)
	if err == sql.ErrNoRows {
		return domain.Alert{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.Alert{}, err
	}
	if current == "acknowledged" {
		return domain.Alert{}, fmt.Errorf("%w: alert already acknowledged", domain.ErrConflict)
	}
	if _, err := tx.ExecContext(ctx, "UPDATE alerts SET status='acknowledged',acknowledged_by=?,acknowledged_at_ms=?,acknowledgement_note=? WHERE id=?", user.ID, now, note, id); err != nil {
		return domain.Alert{}, err
	}
	a, err := scanAlert(tx.QueryRowContext(ctx, "SELECT id,vin,code,title,severity,status,occurred_at_ms,acknowledged_by,acknowledged_at_ms,acknowledgement_note FROM alerts WHERE id=?", id))
	if err != nil {
		return domain.Alert{}, err
	}
	return a, tx.Commit()
}
