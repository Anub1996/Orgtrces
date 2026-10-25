-- Orgtraces Database Schema
-- Designed for 5,000+ concurrent users with TimescaleDB for time-series data

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Organizations (multi-tenant)
CREATE TABLE organizations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    domain VARCHAR(255) UNIQUE,
    plan VARCHAR(50) NOT NULL DEFAULT 'starter', -- starter, professional, enterprise
    max_users INT NOT NULL DEFAULT 30,
    settings JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Managers / Admins (dashboard users)
CREATE TABLE managers (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    email VARCHAR(255) NOT NULL UNIQUE,
    name VARCHAR(255) NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    role VARCHAR(50) NOT NULL DEFAULT 'manager', -- admin, manager, viewer
    is_active BOOLEAN NOT NULL DEFAULT true,
    last_login TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_managers_org ON managers(org_id);
CREATE INDEX idx_managers_email ON managers(email);

-- Employees (monitored users)
CREATE TABLE employees (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    machine_id VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    email VARCHAR(255),
    department VARCHAR(255) DEFAULT 'General',
    agent_version VARCHAR(50),
    status VARCHAR(50) NOT NULL DEFAULT 'offline', -- online, idle, offline, locked, break
    last_seen TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(org_id, machine_id)
);
CREATE INDEX idx_employees_org ON employees(org_id);
CREATE INDEX idx_employees_status ON employees(org_id, status);
CREATE INDEX idx_employees_machine ON employees(machine_id);

-- Sessions (login/logout per employee per day)
CREATE TABLE sessions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    employee_id UUID NOT NULL REFERENCES employees(id) ON DELETE CASCADE,
    login_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    logout_at TIMESTAMPTZ,
    active_time_sec INT NOT NULL DEFAULT 0,
    idle_time_sec INT NOT NULL DEFAULT 0,
    break_time_sec INT NOT NULL DEFAULT 0,
    lock_count INT NOT NULL DEFAULT 0,
    ip_address INET,
    agent_version VARCHAR(50)
);
CREATE INDEX idx_sessions_employee ON sessions(employee_id);
CREATE INDEX idx_sessions_login ON sessions(login_at DESC);
CREATE INDEX idx_sessions_active ON sessions(employee_id, login_at DESC) WHERE logout_at IS NULL;

-- Activities (app usage, URL visits — high-volume, time-series)
CREATE TABLE activities (
    id UUID DEFAULT uuid_generate_v4(),
    employee_id UUID NOT NULL REFERENCES employees(id) ON DELETE CASCADE,
    session_id UUID REFERENCES sessions(id) ON DELETE SET NULL,
    type VARCHAR(50) NOT NULL, -- app_switch, url_visit, file_access, keystroke_summary, mouse_summary
    app_name VARCHAR(512),
    window_title VARCHAR(1024),
    url VARCHAR(2048),
    duration_sec INT NOT NULL DEFAULT 0,
    metadata JSONB DEFAULT '{}',
    timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (id, timestamp)
);
CREATE INDEX idx_activities_employee ON activities(employee_id, timestamp DESC);
CREATE INDEX idx_activities_type ON activities(type, timestamp DESC);

-- Screenshots (metadata only — binary stored in S3)
CREATE TABLE screenshots (
    id UUID DEFAULT uuid_generate_v4(),
    employee_id UUID NOT NULL REFERENCES employees(id) ON DELETE CASCADE,
    session_id UUID REFERENCES sessions(id) ON DELETE SET NULL,
    storage_key VARCHAR(512) NOT NULL,
    file_size BIGINT NOT NULL DEFAULT 0,
    width INT,
    height INT,
    captured_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (id, captured_at)
);
CREATE INDEX idx_screenshots_employee ON screenshots(employee_id, captured_at DESC);

-- Alerts
CREATE TABLE alerts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    employee_id UUID NOT NULL REFERENCES employees(id) ON DELETE CASCADE,
    type VARCHAR(100) NOT NULL, -- idle_exceeded, no_person, multiple_persons, phone_detected, dlp_violation, unauthorized_app
    severity VARCHAR(20) NOT NULL DEFAULT 'medium', -- low, medium, high, critical
    message TEXT NOT NULL,
    data JSONB DEFAULT '{}',
    resolved BOOLEAN NOT NULL DEFAULT false,
    resolved_by UUID REFERENCES managers(id),
    resolved_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_alerts_org ON alerts(org_id, created_at DESC);
CREATE INDEX idx_alerts_unresolved ON alerts(org_id, resolved, created_at DESC) WHERE NOT resolved;
CREATE INDEX idx_alerts_employee ON alerts(employee_id, created_at DESC);

-- Wellness logs
CREATE TABLE wellness_logs (
    id UUID DEFAULT uuid_generate_v4(),
    employee_id UUID NOT NULL REFERENCES employees(id) ON DELETE CASCADE,
    type VARCHAR(50) NOT NULL, -- break_reminder, break_taken, break_skipped
    worked_mins INT NOT NULL DEFAULT 0,
    break_mins INT NOT NULL DEFAULT 0,
    timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (id, timestamp)
);
CREATE INDEX idx_wellness_employee ON wellness_logs(employee_id, timestamp DESC);

-- DLP events
CREATE TABLE dlp_events (
    id UUID DEFAULT uuid_generate_v4(),
    employee_id UUID NOT NULL REFERENCES employees(id) ON DELETE CASCADE,
    type VARCHAR(100) NOT NULL, -- clipboard_blocked, printscreen_blocked, file_transfer_blocked, usb_blocked
    details TEXT,
    blocked BOOLEAN NOT NULL DEFAULT true,
    timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (id, timestamp)
);
CREATE INDEX idx_dlp_employee ON dlp_events(employee_id, timestamp DESC);

-- Policies (per-org or per-department monitoring rules)
CREATE TABLE policies (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    department VARCHAR(255), -- NULL = applies to all
    name VARCHAR(255) NOT NULL,
    type VARCHAR(100) NOT NULL, -- idle_threshold, screenshot_interval, wellness_interval, blocked_apps, blocked_urls, dlp_rules
    config JSONB NOT NULL DEFAULT '{}',
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_policies_org ON policies(org_id);

-- Agent keys (for authenticating desktop agents)
CREATE TABLE agent_keys (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    key_hash VARCHAR(255) NOT NULL,
    label VARCHAR(255),
    is_active BOOLEAN NOT NULL DEFAULT true,
    last_used TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_agent_keys_org ON agent_keys(org_id);

-- Convert high-volume tables to TimescaleDB hypertables (if extension available)
-- Uncomment these if using TimescaleDB:
-- SELECT create_hypertable('activities', 'timestamp', if_not_exists => TRUE);
-- SELECT create_hypertable('screenshots', 'captured_at', if_not_exists => TRUE);
-- SELECT create_hypertable('wellness_logs', 'timestamp', if_not_exists => TRUE);
-- SELECT create_hypertable('dlp_events', 'timestamp', if_not_exists => TRUE);

-- Seed: default organization for development
INSERT INTO organizations (id, name, domain, plan, max_users)
VALUES ('a0000000-0000-0000-0000-000000000001', 'Demo Corp', 'demo.orgtraces.com', 'professional', 100);

-- Seed: default admin (password: admin123 — change immediately)
INSERT INTO managers (id, org_id, email, name, password_hash, role)
VALUES (
    'b0000000-0000-0000-0000-000000000001',
    'a0000000-0000-0000-0000-000000000001',
    'admin@demo.orgtraces.com',
    'Admin User',
    crypt('admin123', gen_salt('bf', 12)),
    'admin'
);
