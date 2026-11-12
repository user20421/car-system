CREATE TABLE IF NOT EXISTS users (
  id CHAR(36) PRIMARY KEY,
  username VARCHAR(128) NOT NULL UNIQUE,
  email VARCHAR(255) NULL,
  display_name VARCHAR(128) NOT NULL,
  roles JSON NOT NULL,
  password_hash VARCHAR(255) NOT NULL,
  created_at DATETIME(6) NOT NULL
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS sessions (
  id CHAR(36) PRIMARY KEY,
  user_id CHAR(36) NOT NULL,
  revoked_at DATETIME(6) NULL,
  created_at DATETIME(6) NOT NULL,
  CONSTRAINT fk_sessions_user FOREIGN KEY (user_id) REFERENCES users(id)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS refresh_tokens (
  id CHAR(36) PRIMARY KEY,
  session_id CHAR(36) NOT NULL,
  user_id CHAR(36) NOT NULL,
  token_hash CHAR(43) NOT NULL UNIQUE,
  expires_at DATETIME(6) NOT NULL,
  used_at DATETIME(6) NULL,
  revoked_at DATETIME(6) NULL,
  replaced_by CHAR(36) NULL,
  created_at DATETIME(6) NOT NULL,
  INDEX idx_refresh_session (session_id),
  CONSTRAINT fk_refresh_session FOREIGN KEY (session_id) REFERENCES sessions(id)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS vehicles (
  vin CHAR(17) PRIMARY KEY,
  display_name VARCHAR(128) NOT NULL,
  model VARCHAR(128) NOT NULL,
  model_year INT NOT NULL,
  owner_user_id CHAR(36) NOT NULL,
  firmware_version VARCHAR(64) NULL,
  created_at DATETIME(6) NOT NULL,
  CONSTRAINT fk_vehicle_owner FOREIGN KEY (owner_user_id) REFERENCES users(id)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS vehicle_shares (
  vin CHAR(17) NOT NULL,
  user_id CHAR(36) NOT NULL,
  access_role ENUM('driver','viewer') NOT NULL,
  created_at DATETIME(6) NOT NULL,
  PRIMARY KEY (vin,user_id),
  CONSTRAINT fk_share_vehicle FOREIGN KEY (vin) REFERENCES vehicles(vin),
  CONSTRAINT fk_share_user FOREIGN KEY (user_id) REFERENCES users(id)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS telemetry (
  id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  message_id VARCHAR(128) NOT NULL,
  vin CHAR(17) NOT NULL,
  device_boot_id VARCHAR(128) NOT NULL,
  sequence BIGINT UNSIGNED NOT NULL,
  occurred_at_ms BIGINT NOT NULL,
  received_at_ms BIGINT NOT NULL,
  payload JSON NOT NULL,
  UNIQUE KEY uq_telemetry_message (vin,message_id),
  UNIQUE KEY uq_telemetry_sequence (vin,device_boot_id,sequence),
  INDEX idx_telemetry_time (vin,occurred_at_ms,id),
  CONSTRAINT fk_telemetry_vehicle FOREIGN KEY (vin) REFERENCES vehicles(vin)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS telemetry_batches (
  batch_id VARCHAR(128) NOT NULL,
  vin CHAR(17) NOT NULL,
  created_at_ms BIGINT NOT NULL,
  PRIMARY KEY (vin,batch_id)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS commands (
  command_id CHAR(36) PRIMARY KEY,
  vin CHAR(17) NOT NULL,
  requested_by CHAR(36) NOT NULL,
  idempotency_key VARCHAR(128) NOT NULL,
  request_hash BINARY(32) NOT NULL,
  expires_at_ms BIGINT NOT NULL,
  type VARCHAR(32) NOT NULL,
  params JSON NOT NULL,
  status ENUM('pending','sent','received','succeeded','failed','expired') NOT NULL,
  error JSON NULL,
  created_at_ms BIGINT NOT NULL,
  updated_at_ms BIGINT NOT NULL,
  UNIQUE KEY uq_command_idempotency (requested_by,vin,idempotency_key),
  INDEX idx_commands_vin (vin,created_at_ms),
  CONSTRAINT fk_command_vehicle FOREIGN KEY (vin) REFERENCES vehicles(vin),
  CONSTRAINT fk_command_user FOREIGN KEY (requested_by) REFERENCES users(id)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS outbox (
  id CHAR(36) PRIMARY KEY,
  event_type VARCHAR(64) NOT NULL,
  aggregate_id CHAR(36) NOT NULL,
  payload JSON NOT NULL,
  created_at_ms BIGINT NOT NULL,
  published_at_ms BIGINT NULL,
  attempts INT NOT NULL DEFAULT 0,
  last_error VARCHAR(1000) NULL,
  INDEX idx_outbox_unpublished (published_at_ms,created_at_ms)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS command_dispatch (
  command_id CHAR(36) PRIMARY KEY,
  event_id CHAR(36) NOT NULL UNIQUE,
  processed_at_ms BIGINT NOT NULL
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS alerts (
  id CHAR(36) PRIMARY KEY,
  vin CHAR(17) NOT NULL,
  code VARCHAR(64) NOT NULL,
  title VARCHAR(255) NOT NULL,
  severity ENUM('info','warning','critical') NOT NULL,
  status ENUM('open','acknowledged') NOT NULL DEFAULT 'open',
  occurred_at_ms BIGINT NOT NULL,
  acknowledged_by CHAR(36) NULL,
  acknowledged_at_ms BIGINT NULL,
  acknowledgement_note VARCHAR(1000) NULL,
  UNIQUE KEY uq_alert_occurrence (vin,code,occurred_at_ms),
  INDEX idx_alerts_status (status,occurred_at_ms),
  CONSTRAINT fk_alert_vehicle FOREIGN KEY (vin) REFERENCES vehicles(vin)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS device_configs (
  vin CHAR(17) PRIMARY KEY,
  config_version BIGINT NOT NULL,
  telemetry_interval_ms INT NOT NULL,
  config_poll_interval_ms INT NOT NULL,
  updated_at_ms BIGINT NOT NULL,
  CONSTRAINT fk_config_vehicle FOREIGN KEY (vin) REFERENCES vehicles(vin)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS ota_releases (
  id CHAR(36) PRIMARY KEY,
  model VARCHAR(128) NOT NULL,
  version VARCHAR(64) NOT NULL,
  url VARCHAR(2048) NOT NULL,
  sha256 CHAR(64) NOT NULL,
  size_bytes BIGINT NOT NULL,
  mandatory BOOLEAN NOT NULL,
  release_notes TEXT NULL,
  released_at_ms BIGINT NOT NULL,
  UNIQUE KEY uq_ota_model_version (model,version)
) ENGINE=InnoDB;
