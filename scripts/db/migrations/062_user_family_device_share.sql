SET search_path TO public;

CREATE TABLE IF NOT EXISTS public.user_family (
  id BIGSERIAL PRIMARY KEY,
  owner_user_id BIGINT NOT NULL REFERENCES public.users(id) ON DELETE CASCADE,
  name VARCHAR(64) NOT NULL DEFAULT '',
  status SMALLINT NOT NULL DEFAULT 1,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT user_family_status_check CHECK (status IN (1, 2))
);

CREATE UNIQUE INDEX IF NOT EXISTS uk_user_family_owner_active
  ON public.user_family (owner_user_id)
  WHERE status = 1;

DROP TRIGGER IF EXISTS trg_user_family_set_updated_at ON public.user_family;
CREATE TRIGGER trg_user_family_set_updated_at
BEFORE UPDATE ON public.user_family
FOR EACH ROW EXECUTE FUNCTION public.set_updated_at();

CREATE TABLE IF NOT EXISTS public.user_family_member (
  id BIGSERIAL PRIMARY KEY,
  family_id BIGINT NOT NULL REFERENCES public.user_family(id) ON DELETE CASCADE,
  user_id BIGINT NOT NULL REFERENCES public.users(id) ON DELETE CASCADE,
  role VARCHAR(16) NOT NULL DEFAULT 'member',
  status SMALLINT NOT NULL DEFAULT 1,
  joined_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  invited_by BIGINT NOT NULL DEFAULT 0,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT user_family_member_role_check CHECK (role IN ('owner', 'super_admin', 'member')),
  CONSTRAINT user_family_member_status_check CHECK (status IN (1, 2))
);

CREATE UNIQUE INDEX IF NOT EXISTS uk_user_family_member_active
  ON public.user_family_member (family_id, user_id)
  WHERE status = 1;
CREATE INDEX IF NOT EXISTS idx_user_family_member_user
  ON public.user_family_member (user_id, status, family_id);

DROP TRIGGER IF EXISTS trg_user_family_member_set_updated_at ON public.user_family_member;
CREATE TRIGGER trg_user_family_member_set_updated_at
BEFORE UPDATE ON public.user_family_member
FOR EACH ROW EXECUTE FUNCTION public.set_updated_at();

CREATE TABLE IF NOT EXISTS public.user_family_invite (
  id BIGSERIAL PRIMARY KEY,
  family_id BIGINT NOT NULL REFERENCES public.user_family(id) ON DELETE CASCADE,
  invite_code VARCHAR(64) NOT NULL,
  target_user_id BIGINT NULL REFERENCES public.users(id) ON DELETE SET NULL,
  target_account VARCHAR(128) NOT NULL DEFAULT '',
  role VARCHAR(16) NOT NULL DEFAULT 'member',
  status VARCHAR(16) NOT NULL DEFAULT 'pending',
  expires_at TIMESTAMP NULL,
  created_by BIGINT NOT NULL DEFAULT 0,
  remark VARCHAR(255) NOT NULL DEFAULT '',
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT user_family_invite_role_check CHECK (role IN ('super_admin', 'member')),
  CONSTRAINT user_family_invite_status_check CHECK (status IN ('pending', 'accepted', 'expired', 'cancelled'))
);

CREATE UNIQUE INDEX IF NOT EXISTS uk_user_family_invite_code
  ON public.user_family_invite (invite_code);
CREATE INDEX IF NOT EXISTS idx_user_family_invite_family_status
  ON public.user_family_invite (family_id, status, expires_at);

DROP TRIGGER IF EXISTS trg_user_family_invite_set_updated_at ON public.user_family_invite;
CREATE TRIGGER trg_user_family_invite_set_updated_at
BEFORE UPDATE ON public.user_family_invite
FOR EACH ROW EXECUTE FUNCTION public.set_updated_at();

CREATE TABLE IF NOT EXISTS public.user_device_share (
  id BIGSERIAL PRIMARY KEY,
  family_id BIGINT NOT NULL REFERENCES public.user_family(id) ON DELETE CASCADE,
  device_id BIGINT NOT NULL REFERENCES public.device(id) ON DELETE CASCADE,
  device_sn VARCHAR(64) NOT NULL,
  device_name VARCHAR(64) NOT NULL DEFAULT '',
  owner_user_id BIGINT NOT NULL REFERENCES public.users(id) ON DELETE CASCADE,
  shared_user_id BIGINT NOT NULL REFERENCES public.users(id) ON DELETE CASCADE,
  target_account VARCHAR(128) NOT NULL DEFAULT '',
  invite_code VARCHAR(64) NOT NULL DEFAULT '',
  share_type VARCHAR(16) NOT NULL DEFAULT 'permanent',
  permission_level VARCHAR(16) NOT NULL DEFAULT 'view_only',
  permission_payload JSONB NOT NULL DEFAULT '{}'::jsonb,
  start_at TIMESTAMP NULL,
  end_at TIMESTAMP NULL,
  status VARCHAR(16) NOT NULL DEFAULT 'pending',
  confirmed_at TIMESTAMP NULL,
  revoked_at TIMESTAMP NULL,
  created_by BIGINT NOT NULL DEFAULT 0,
  remark VARCHAR(255) NOT NULL DEFAULT '',
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT user_device_share_type_check CHECK (share_type IN ('permanent', 'temporary', 'time_window')),
  CONSTRAINT user_device_share_permission_check CHECK (permission_level IN ('full_control', 'partial_control', 'view_only')),
  CONSTRAINT user_device_share_status_check CHECK (status IN ('pending', 'active', 'revoked', 'expired', 'quit'))
);

CREATE UNIQUE INDEX IF NOT EXISTS uk_user_device_share_active
  ON public.user_device_share (device_id, shared_user_id)
  WHERE status IN ('pending', 'active');
CREATE UNIQUE INDEX IF NOT EXISTS uk_user_device_share_invite_code
  ON public.user_device_share (invite_code)
  WHERE invite_code <> '';
CREATE INDEX IF NOT EXISTS idx_user_device_share_owner
  ON public.user_device_share (owner_user_id, status, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_user_device_share_shared
  ON public.user_device_share (shared_user_id, status, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_user_device_share_device
  ON public.user_device_share (device_id, status, end_at);
CREATE INDEX IF NOT EXISTS idx_user_device_share_status_time
  ON public.user_device_share (status, end_at);

DROP TRIGGER IF EXISTS trg_user_device_share_set_updated_at ON public.user_device_share;
CREATE TRIGGER trg_user_device_share_set_updated_at
BEFORE UPDATE ON public.user_device_share
FOR EACH ROW EXECUTE FUNCTION public.set_updated_at();

CREATE TABLE IF NOT EXISTS public.user_device_share_log (
  id BIGSERIAL PRIMARY KEY,
  share_id BIGINT NOT NULL REFERENCES public.user_device_share(id) ON DELETE CASCADE,
  family_id BIGINT NOT NULL REFERENCES public.user_family(id) ON DELETE CASCADE,
  device_id BIGINT NOT NULL REFERENCES public.device(id) ON DELETE CASCADE,
  device_sn VARCHAR(64) NOT NULL,
  op_type VARCHAR(32) NOT NULL DEFAULT '',
  op_content TEXT NOT NULL DEFAULT '',
  operator_user_id BIGINT NOT NULL DEFAULT 0,
  operator_role VARCHAR(16) NOT NULL DEFAULT '',
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_user_device_share_log_share
  ON public.user_device_share_log (share_id, created_at DESC);
