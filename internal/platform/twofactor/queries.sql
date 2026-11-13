-- name: FindTwoFactorStatus :one
SELECT
    two_factor_enabled,
    two_factor_secret,
    two_factor_secret_enc,
    two_factor_key_version,
    two_factor_last_step
FROM users
WHERE public_id = sqlc.arg('public_id')
  AND deleted_at IS NULL
LIMIT 1;

-- name: SetTwoFactorSecret :execrows
UPDATE users
SET two_factor_secret = sqlc.arg('two_factor_secret'),
    two_factor_secret_enc = sqlc.arg('two_factor_secret_enc'),
    two_factor_key_version = sqlc.arg('two_factor_key_version'),
    two_factor_enabled = false,
    two_factor_confirmed_at = NULL,
    two_factor_last_step = 0,
    updated_at = now()
WHERE public_id = sqlc.arg('public_id')
  AND deleted_at IS NULL;

-- name: SealTwoFactorSecret :execrows
UPDATE users
SET two_factor_secret_enc = sqlc.arg('two_factor_secret_enc'),
    two_factor_key_version = sqlc.arg('two_factor_key_version'),
    two_factor_secret = '',
    updated_at = now()
WHERE public_id = sqlc.arg('public_id')
  AND deleted_at IS NULL;

-- name: EnableTwoFactor :execrows
UPDATE users
SET two_factor_enabled = true,
    two_factor_confirmed_at = now(),
    two_factor_last_step = sqlc.arg('two_factor_last_step'),
    updated_at = now()
WHERE public_id = sqlc.arg('public_id')
  AND deleted_at IS NULL;

-- name: DisableTwoFactor :execrows
UPDATE users
SET two_factor_enabled = false,
    two_factor_secret = '',
    two_factor_secret_enc = ''::bytea,
    two_factor_key_version = 0,
    two_factor_last_step = 0,
    two_factor_confirmed_at = NULL,
    updated_at = now()
WHERE public_id = sqlc.arg('public_id')
  AND deleted_at IS NULL;

-- name: ConsumeTwoFactorStep :execrows
UPDATE users
SET two_factor_last_step = sqlc.arg('two_factor_last_step'),
    updated_at = now()
WHERE public_id = sqlc.arg('public_id')
  AND two_factor_last_step < sqlc.arg('two_factor_last_step')
  AND deleted_at IS NULL;

-- name: DeleteRecoveryCodes :exec
DELETE FROM user_recovery_codes
WHERE user_id = (
    SELECT u.id FROM users u
    WHERE u.public_id = sqlc.arg('public_id')
      AND u.deleted_at IS NULL
  );

-- name: InsertRecoveryCodes :exec
INSERT INTO user_recovery_codes (user_id, code_hash)
SELECT u.id, h
FROM users u
CROSS JOIN unnest(sqlc.arg('code_hashes')::text[]) AS h
WHERE u.public_id = sqlc.arg('public_id')
  AND u.deleted_at IS NULL;

-- name: ListUnusedRecoveryCodes :many
SELECT c.id, c.code_hash
FROM user_recovery_codes c
JOIN users u ON u.id = c.user_id
WHERE u.public_id = sqlc.arg('public_id')
  AND u.deleted_at IS NULL
  AND c.used_at IS NULL
ORDER BY c.id;

-- name: MarkRecoveryCodeUsed :execrows
UPDATE user_recovery_codes
SET used_at = now()
WHERE id = sqlc.arg('id')
  AND used_at IS NULL;

-- name: ReplaceRecoveryCodeHash :execrows
UPDATE user_recovery_codes
SET code_hash = sqlc.arg('code_hash')
WHERE id = sqlc.arg('id');

-- name: ListLegacyTwoFactorSecrets :many
SELECT public_id, two_factor_secret
FROM users
WHERE two_factor_secret <> ''
  AND two_factor_secret_enc = ''::bytea
  AND deleted_at IS NULL
ORDER BY id
LIMIT sqlc.arg('row_limit');
