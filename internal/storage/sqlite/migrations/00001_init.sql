-- +goose Up
CREATE TABLE users (
                       id            INTEGER PRIMARY KEY AUTOINCREMENT,
                       email         TEXT    NOT NULL UNIQUE,
                       password_hash TEXT    NOT NULL,
                       created_at    TEXT    NOT NULL
);

CREATE TABLE sessions (
                          token      TEXT    PRIMARY KEY,
                          user_id    INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
                          created_at TEXT    NOT NULL,
                          expires_at TEXT    NOT NULL
);
CREATE INDEX idx_sessions_user_id ON sessions(user_id);

CREATE TABLE workspaces (
                            id          INTEGER PRIMARY KEY AUTOINCREMENT,
                            name        TEXT    NOT NULL,
                            is_personal INTEGER NOT NULL DEFAULT 0 CHECK (is_personal IN (0, 1)),
                            created_by  INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
                            created_at  TEXT    NOT NULL
);
CREATE UNIQUE INDEX uq_workspaces_personal ON workspaces(created_by) WHERE is_personal = 1;

CREATE TABLE workspace_members (
                                   workspace_id INTEGER NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
                                   user_id      INTEGER NOT NULL REFERENCES users(id)      ON DELETE CASCADE,
                                   role         TEXT    NOT NULL CHECK (role IN ('owner', 'admin', 'member')),
                                   created_at   TEXT    NOT NULL,
                                   PRIMARY KEY (workspace_id, user_id)
);
CREATE INDEX idx_members_user_id ON workspace_members(user_id);

CREATE TABLE projects (
                          id           INTEGER PRIMARY KEY AUTOINCREMENT,
                          workspace_id INTEGER NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
                          name         TEXT    NOT NULL,
                          description  TEXT    NOT NULL DEFAULT '',
                          created_at   TEXT    NOT NULL,
                          updated_at   TEXT    NOT NULL
);
CREATE INDEX idx_projects_workspace_id ON projects(workspace_id);

CREATE TABLE tasks (
                       id          INTEGER PRIMARY KEY AUTOINCREMENT,
                       project_id  INTEGER NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
                       title       TEXT    NOT NULL,
                       description TEXT    NOT NULL DEFAULT '',
                       status      TEXT    NOT NULL DEFAULT 'todo' CHECK (status IN ('todo', 'in_progress', 'done')),
                       priority    TEXT    NOT NULL DEFAULT 'medium' CHECK (priority IN ('low', 'medium', 'high')),
                       due_date    TEXT,
                       created_at  TEXT    NOT NULL,
                       updated_at  TEXT    NOT NULL
);
CREATE INDEX idx_tasks_project_status ON tasks(project_id, status);

-- +goose Down
DROP TABLE tasks;
DROP TABLE projects;
DROP TABLE workspace_members;
DROP TABLE workspaces;
DROP TABLE sessions;
DROP TABLE users;