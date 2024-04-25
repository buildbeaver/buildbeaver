package migrations

// DialectTemplate is used as the templating control for differing SQL syntax between our supported databases
type DialectTemplate struct {
	Binary            string
	IntegerPrimaryKey string
}

// MigrationSet provides a set of migrations that can be applied to a database.
type MigrationSet []MigrationData

// MigrationData provides the data for a single migration, including Up and Down SQL.
// Templated values are supported and will be substituted for database-specific values
// before the migrations are applied.
type MigrationData struct {
	SequenceNumber int64
	Name           string
	UpSQL          string
	DownSQL        string
}

// BuildBeaverServerMigrations is the set of migrations to set up the database for the BuildBeaver server.
var BuildBeaverServerMigrations = MigrationSet{
	{
		SequenceNumber: 1,
		Name:           "create_identities",
		UpSQL: `CREATE TABLE IF NOT EXISTS identities
				(
					identity_id text NOT NULL PRIMARY KEY,
					identity_created_at timestamp without time zone NOT NULL,
					identity_owner_resource_id text NOT NULL
				);
				CREATE UNIQUE INDEX IF NOT EXISTS identities_owner_resource_id_index ON identities(identity_owner_resource_id);`,
		DownSQL: `DROP INDEX identities_owner_resource_id_index;
				  DROP TABLE identities;`,
	},
	{
		SequenceNumber: 2,
		Name:           "create_legal_entities",
		UpSQL: `CREATE TABLE IF NOT EXISTS legal_entities
				(
					legal_entity_id text NOT NULL PRIMARY KEY,
					legal_entity_name text NOT NULL,
					legal_entity_created_at timestamp without time zone NOT NULL,
					legal_entity_updated_at timestamp without time zone NOT NULL,
					legal_entity_deleted_at timestamp without time zone,
					legal_entity_etag text NOT NULL,
					legal_entity_type text NOT NULL,
					legal_entity_legal_name text NOT NULL,
					legal_entity_email_address text NOT NULL,
					legal_entity_external_id text,
					legal_entity_external_metadata text
				);
				CREATE UNIQUE INDEX IF NOT EXISTS legal_entities_external_id_unique_index ON legal_entities(legal_entity_external_id)
				WHERE legal_entity_deleted_at IS NULL;
				CREATE UNIQUE INDEX IF NOT EXISTS legal_entities_name_unique_index ON legal_entities(legal_entity_name)
				WHERE legal_entity_deleted_at IS NULL;
				CREATE UNIQUE INDEX IF NOT EXISTS legal_entities_created_at_id_desc_unique_index ON legal_entities(
					legal_entity_created_at DESC,
					legal_entity_id DESC);`,
		DownSQL: `DROP TABLE legal_entities;`,
	},
	{
		SequenceNumber: 3,
		Name:           "create_repos",
		UpSQL: `CREATE TABLE IF NOT EXISTS repos
				(
					repo_id text NOT NULL PRIMARY KEY,
					repo_created_at timestamp without time zone NOT NULL,
					repo_updated_at timestamp without time zone NOT NULL,
					repo_deleted_at timestamp without time zone,
					repo_etag text NOT NULL,
					repo_legal_entity_id text NOT NULL REFERENCES legal_entities (legal_entity_id) ON UPDATE NO ACTION ON DELETE NO ACTION,
					repo_name text NOT NULL,
					repo_description text NOT NULL,
					repo_ssh_url text NOT NULL,
					repo_http_url text NOT NULL,
					repo_link text NOT NULL,
					repo_default_branch text NOT NULL,
					repo_private bool NOT NULL,
					repo_enabled bool NOT NULL,
					repo_build_counter integer NOT NULL DEFAULT 0,
					repo_external_id text,
					repo_external_metadata text
				);
				CREATE UNIQUE INDEX IF NOT EXISTS repos_external_id_unique_index ON repos(repo_external_id)
				WHERE repo_deleted_at IS NULL;
				CREATE UNIQUE INDEX IF NOT EXISTS repos_created_at_id_desc_unique_index ON repos(
					repo_created_at DESC,
					repo_id DESC);`,
		DownSQL: `DROP TABLE repos;`,
	},
	{
		SequenceNumber: 4,
		Name:           "create_commits",
		UpSQL: `CREATE TABLE IF NOT EXISTS commits
				(
					commit_id text NOT NULL PRIMARY KEY,
					commit_created_at timestamp without time zone NOT NULL,
					commit_repo_id text NOT NULL REFERENCES repos (repo_id) ON UPDATE NO ACTION ON DELETE NO ACTION,
					commit_config {{ .Binary}},
					commit_config_type text,
					commit_sha text NOT NULL,
					commit_message text NOT NULL,
					commit_author_id text REFERENCES legal_entities (legal_entity_id) ON UPDATE NO ACTION ON DELETE NO ACTION,
					commit_author_name text,
					commit_author_email text,
					commit_committer_id text REFERENCES legal_entities (legal_entity_id) ON UPDATE NO ACTION ON DELETE NO ACTION,
					commit_committer_name text,
					commit_committer_email text,
					commit_link text NOT NULL
				);
				CREATE UNIQUE INDEX IF NOT EXISTS commits_sha_unique_index ON commits(commit_repo_id,commit_sha);
				CREATE UNIQUE INDEX IF NOT EXISTS commits_created_at_id_desc_unique_index ON commits(
					commit_created_at DESC,
					commit_id DESC);`,
		DownSQL: `DROP TABLE commits;`,
	},
	{
		SequenceNumber: 5,
		Name:           "create_log_descriptors",
		UpSQL: `CREATE TABLE IF NOT EXISTS log_descriptors
				(
					log_descriptor_id text NOT NULL PRIMARY KEY,
					log_descriptor_parent_log_id text REFERENCES log_descriptors (log_descriptor_id) ON UPDATE NO ACTION ON DELETE NO ACTION,
					log_descriptor_created_at timestamp without time zone NOT NULL,
					log_descriptor_updated_at timestamp without time zone NOT NULL,
					log_descriptor_resource_id text NOT NULL,
					log_descriptor_sealed bool NOT NULL,
					log_descriptor_size_bytes integer NOT NULL default 0,
					log_descriptor_etag text NOT NULL
				);
				CREATE INDEX IF NOT EXISTS log_descriptors_resource_id_index ON log_descriptors(log_descriptor_resource_id);
				CREATE INDEX IF NOT EXISTS log_descriptors_parent_log_id_index ON log_descriptors(log_descriptor_parent_log_id);
				CREATE UNIQUE INDEX IF NOT EXISTS log_descriptors_created_at_id_desc_unique_index ON log_descriptors(
					log_descriptor_created_at DESC,
					log_descriptor_id DESC);`,
		DownSQL: `DROP TABLE log_descriptors;`,
	},
	{
		SequenceNumber: 6,
		Name:           "create_builds",
		UpSQL: `CREATE TABLE IF NOT EXISTS builds
				(
					build_id text NOT NULL PRIMARY KEY,
					build_name text NOT NULL,
					build_repo_id text NOT NULL REFERENCES repos (repo_id) ON UPDATE NO ACTION ON DELETE NO ACTION,
					build_created_at timestamp without time zone NOT NULL,
					build_updated_at timestamp without time zone NOT NULL,
					build_deleted_at timestamp without time zone,
					build_commit_id text NOT NULL REFERENCES commits (commit_id) ON UPDATE NO ACTION ON DELETE NO ACTION,
					build_log_descriptor_id text NOT NULL REFERENCES log_descriptors (log_descriptor_id) ON UPDATE NO ACTION ON DELETE NO ACTION,
					build_ref text NOT NULL,
					build_etag text NOT NULL,
					build_status text NOT NULL,
					build_error text,
					build_opts text,
					build_timings text NOT NULL
				);
				CREATE UNIQUE INDEX IF NOT EXISTS builds_build_name_unique_index ON builds(
					build_repo_id,
					build_name)
				WHERE build_deleted_at IS NULL;
				CREATE UNIQUE INDEX IF NOT EXISTS builds_created_at_id_desc_unique_index ON builds(
					build_created_at DESC,
					build_id DESC);`,
		DownSQL: `DROP TABLE builds;`,
	},
	{
		SequenceNumber: 7,
		Name:           "create_runners",
		UpSQL: `CREATE TABLE IF NOT EXISTS runners
				(
					runner_id text NOT NULL PRIMARY KEY,
					runner_name text NOT NULL,
					runner_legal_entity_id text NOT NULL REFERENCES legal_entities (legal_entity_id) ON UPDATE NO ACTION ON DELETE NO ACTION,
					runner_created_at timestamp without time zone NOT NULL,
					runner_updated_at timestamp without time zone NOT NULL,
					runner_deleted_at timestamp without time zone,
					runner_etag text NOT NULL,
					runner_software_version text NOT NULL
				);
				CREATE INDEX IF NOT EXISTS runners_legal_entity_id_index ON runners(runner_legal_entity_id);
				CREATE UNIQUE INDEX IF NOT EXISTS runners_name_unique_index ON runners(
					runner_name,
					runner_legal_entity_id)
				WHERE runner_deleted_at IS NULL;
				CREATE UNIQUE INDEX IF NOT EXISTS runners_created_at_id_desc_unique_index ON runners(
					runner_created_at DESC,
					runner_id DESC);`,
		DownSQL: `DROP TABLE runners;`,
	},
	{
		SequenceNumber: 8,
		Name:           "create_jobs",
		UpSQL: `CREATE TABLE IF NOT EXISTS jobs
				(
					job_id text NOT NULL PRIMARY KEY,
					job_name text NOT NULL,
					job_build_id text NOT NULL REFERENCES builds (build_id) ON UPDATE NO ACTION ON DELETE NO ACTION,
					job_created_at timestamp without time zone NOT NULL,
					job_updated_at timestamp without time zone NOT NULL,
					job_deleted_at timestamp without time zone,
					job_repo_id text NOT NULL REFERENCES repos (repo_id) ON UPDATE NO ACTION ON DELETE NO ACTION,
					job_commit_id text NOT NULL REFERENCES commits (commit_id) ON UPDATE NO ACTION ON DELETE NO ACTION,
					job_log_descriptor_id text NOT NULL REFERENCES log_descriptors (log_descriptor_id) ON UPDATE NO ACTION ON DELETE NO ACTION,
					job_runner_id text REFERENCES runners (runner_id) ON UPDATE NO ACTION ON DELETE NO ACTION,
					job_ref text NOT NULL,
					job_etag text NOT NULL,
					job_status text NOT NULL,
					job_error text,
					job_description text NOT NULL,
					job_depends text,
					job_type text NOT NULL,
					job_services text,
					job_timings text NOT NULL,
					job_step_execution text NOT NULL
				);
				CREATE UNIQUE INDEX IF NOT EXISTS jobs_job_name_unique_index ON jobs(
					job_build_id,
					job_name)
				WHERE job_deleted_at IS NULL;
				CREATE UNIQUE INDEX IF NOT EXISTS jobs_created_at_id_desc_unique_index ON jobs(
					job_created_at DESC,
					job_id DESC);`,
		DownSQL: `DROP TABLE jobs;`,
	},
	{
		SequenceNumber: 9,
		Name:           "create_jobs_depend_on_jobs",
		UpSQL: `CREATE TABLE IF NOT EXISTS jobs_depend_on_jobs
				(
				   jobs_depend_on_jobs_id {{ .IntegerPrimaryKey}},
				   jobs_depend_on_jobs_source_job_id text NOT NULL REFERENCES jobs (job_id) ON UPDATE NO ACTION ON DELETE NO ACTION,
				   jobs_depend_on_jobs_target_job_id text NOT NULL REFERENCES jobs (job_id) ON UPDATE NO ACTION ON DELETE NO ACTION
				);`,
		DownSQL: `DROP TABLE jobs_depend_on_jobs;`,
	},
	{
		SequenceNumber: 10,
		Name:           "create_steps",
		UpSQL: `CREATE TABLE IF NOT EXISTS steps
				(
					step_id text NOT NULL PRIMARY KEY,
					step_name text NOT NULL,
					step_job_id text NOT NULL REFERENCES jobs (job_id) ON UPDATE NO ACTION ON DELETE NO ACTION,
					step_created_at timestamp without time zone NOT NULL,
					step_updated_at timestamp without time zone NOT NULL,
					step_deleted_at timestamp without time zone,
					step_repo_id text NOT NULL REFERENCES repos (repo_id) ON UPDATE NO ACTION ON DELETE NO ACTION,
					step_log_descriptor_id text NOT NULL REFERENCES log_descriptors (log_descriptor_id) ON UPDATE NO ACTION ON DELETE NO ACTION,
					step_runner_id text REFERENCES runners (runner_id) ON UPDATE NO ACTION ON DELETE NO ACTION,
					step_indirect_to_step_id text REFERENCES steps (step_id) ON UPDATE NO ACTION ON DELETE NO ACTION,
					step_etag text NOT NULL,
					step_status text NOT NULL,
					step_description text NOT NULL,
					step_depends text,
					step_commands text NOT NULL,
					step_error text,
					step_docker_image text NOT NULL,
					step_docker_image_pull_strategy text NOT NULL,
					step_docker_authentication text,
					step_environment text,
					step_fingerprint_commands text,
					step_fingerprint text,
					step_fingerprint_hash_type text,
					step_artifact_definitions text,
					step_timings text NOT NULL
				);
				CREATE UNIQUE INDEX IF NOT EXISTS steps_step_name_unique_index ON steps(
					step_job_id,
					step_name)
				WHERE step_deleted_at IS NULL;
				CREATE UNIQUE INDEX IF NOT EXISTS steps_created_at_id_desc_unique_index ON steps(
					step_created_at DESC,
					step_id DESC);`,
		DownSQL: `DROP TABLE steps;`,
	},
	{
		SequenceNumber: 11,
		Name:           "create_secrets",
		UpSQL: `CREATE TABLE IF NOT EXISTS secrets
				(
					secret_id text NOT NULL PRIMARY KEY,
					secret_name text NOT NULL,
					secret_repo_id text NOT NULL REFERENCES repos (repo_id) ON UPDATE NO ACTION ON DELETE NO ACTION,
					secret_created_at timestamp without time zone NOT NULL,
					secret_updated_at timestamp without time zone NOT NULL,
					secret_etag text NOT NULL,
					secret_key_encrypted {{ .Binary}} NOT NULL,
					secret_value_encrypted {{ .Binary}} NOT NULL,
					secret_data_key_encrypted {{ .Binary}} NOT NULL,
					secret_is_internal BOOL NOT NULL
				);
				CREATE UNIQUE INDEX IF NOT EXISTS secrets_secret_name_unique_index ON secrets(
					secret_repo_id,
					secret_name);
				CREATE UNIQUE INDEX IF NOT EXISTS secrets_created_at_id_desc_unique_index ON secrets(
					secret_created_at DESC,
					secret_id DESC);`,
		DownSQL: `DROP TABLE secrets;`,
	},
	{
		SequenceNumber: 12,
		Name:           "create_access_control_groups",
		UpSQL: `CREATE TABLE IF NOT EXISTS access_control_groups
				(
					access_control_group_id text NOT NULL PRIMARY KEY,
					access_control_group_created_at timestamp without time zone NOT NULL,
					access_control_group_updated_at timestamp without time zone NOT NULL,
					access_control_group_deleted_at timestamp without time zone,
					access_control_group_etag text NOT NULL,
					access_control_group_name text NOT NULL,
					access_control_group_description text NOT NULL,
					access_control_group_is_internal bool NOT NULL,
					access_control_group_legal_entity_id text NOT NULL REFERENCES legal_entities (legal_entity_id) ON UPDATE NO ACTION ON DELETE NO ACTION
				);
				CREATE UNIQUE INDEX IF NOT EXISTS groups_name_unique_index ON access_control_groups(
					access_control_group_name,
					access_control_group_legal_entity_id)
				WHERE access_control_group_deleted_at IS NULL;
				CREATE UNIQUE INDEX IF NOT EXISTS access_control_groups_created_at_id_desc_unique_index ON access_control_groups(
					access_control_group_created_at DESC,
					access_control_group_id DESC);`,
		DownSQL: `DROP TABLE access_control_groups;`,
	},
	{
		SequenceNumber: 13,
		Name:           "create_access_control_group_memberships",
		UpSQL: `CREATE TABLE IF NOT EXISTS access_control_group_memberships
				(
					access_control_group_membership_id text NOT NULL PRIMARY KEY,
					access_control_group_membership_created_at timestamp without time zone NOT NULL,
					access_control_group_membership_added_by_legal_entity_id text NOT NULL REFERENCES legal_entities (legal_entity_id) ON UPDATE NO ACTION ON DELETE NO ACTION,
					access_control_group_membership_group_id text NOT NULL REFERENCES access_control_groups (access_control_group_id) ON UPDATE NO ACTION ON DELETE NO ACTION,
					access_control_group_membership_member_identity_id text NOT NULL REFERENCES identities (identity_id) ON UPDATE NO ACTION ON DELETE NO ACTION
				);
				CREATE UNIQUE INDEX IF NOT EXISTS group_membership_group_id_identity_id_unique_index ON access_control_group_memberships(
					access_control_group_membership_group_id,
					access_control_group_membership_member_identity_id);
				CREATE UNIQUE INDEX IF NOT EXISTS access_control_group_memberships_created_at_id_desc_unique_index ON access_control_group_memberships(
					access_control_group_membership_created_at DESC,
					access_control_group_membership_id DESC);`,
		DownSQL: `DROP TABLE access_control_group_memberships;`,
	},
	{
		SequenceNumber: 14,
		Name:           "create_access_control_grants",
		UpSQL: `CREATE TABLE IF NOT EXISTS access_control_grants
				(
					access_control_grant_id text NOT NULL PRIMARY KEY,
					access_control_grant_created_at timestamp without time zone NOT NULL,
					access_control_grant_updated_at timestamp without time zone NOT NULL,
					access_control_grant_granted_by_legal_entity_id text NOT NULL REFERENCES legal_entities (legal_entity_id) ON UPDATE NO ACTION ON DELETE NO ACTION,
					access_control_grant_authorized_identity_id text REFERENCES identities (identity_id) ON UPDATE NO ACTION ON DELETE NO ACTION,
					access_control_grant_authorized_group_id text REFERENCES access_control_groups (access_control_group_id) ON UPDATE NO ACTION ON DELETE NO ACTION,
					access_control_grant_operation_name text NOT NULL,
					access_control_grant_operation_resource_kind text NOT NULL,
					access_control_grant_target_resource_id text NOT NULL
				);
				CREATE UNIQUE INDEX IF NOT EXISTS access_control_grants_unique_index ON access_control_grants(
					coalesce(access_control_grant_authorized_identity_id, ''),
					coalesce(access_control_grant_authorized_group_id, ''),
					access_control_grant_operation_name,
					access_control_grant_operation_resource_kind,
					access_control_grant_target_resource_id);
				CREATE UNIQUE INDEX IF NOT EXISTS access_control_grants_created_at_id_desc_unique_index ON access_control_grants(
					access_control_grant_created_at DESC,
					access_control_grant_id DESC);`,
		DownSQL: `DROP TABLE access_control_grants;`,
	},
	{
		SequenceNumber: 15,
		Name:           "create_access_control_ownerships",
		UpSQL: `CREATE TABLE IF NOT EXISTS access_control_ownerships
				(
					access_control_ownership_id text NOT NULL PRIMARY KEY,
					access_control_ownership_updated_at timestamp without time zone NOT NULL,
					access_control_ownership_etag text NOT NULL,
					access_control_ownership_created_at timestamp without time zone NOT NULL,
					access_control_ownership_owner_resource_id text NOT NULL,
					access_control_ownership_owned_resource_id text NOT NULL
				);
				CREATE UNIQUE INDEX IF NOT EXISTS access_control_ownerships_owned_resource_id_unique_index ON access_control_ownerships(
					access_control_ownership_owned_resource_id);
				CREATE UNIQUE INDEX IF NOT EXISTS access_control_ownerships_created_at_id_desc_unique_index ON access_control_ownerships(
					access_control_ownership_created_at DESC,
					access_control_ownership_id DESC);`,
		DownSQL: `DROP TABLE access_control_ownerships;`,
	},
	{
		SequenceNumber: 16,
		Name:           "create_credentials",
		UpSQL: `CREATE TABLE IF NOT EXISTS credentials
				(
					credential_id text NOT NULL PRIMARY KEY,
					credential_created_at timestamp without time zone NOT NULL,
					credential_updated_at timestamp without time zone NOT NULL,
					credential_identity_id text NOT NULL REFERENCES identities (identity_id) ON UPDATE NO ACTION ON DELETE NO ACTION,
					credential_etag text NOT NULL,
					credential_type text NOT NULL,
					credential_is_enabled bool NOT NULL,
					credential_shared_secret_id {{ .Binary}},
					credential_shared_secret_salt {{ .Binary}},
					credential_shared_secret_data_hashed {{ .Binary}},
					credential_github_user_id text,
					credential_client_public_key_asn1_hash_type text,
					credential_client_public_key_asn1_hash text,
					credential_client_public_key_pem text,
					credential_client_certificate_asn1 {{ .Binary}}
				);
				CREATE UNIQUE INDEX IF NOT EXISTS credentials_created_at_id_desc_unique_index ON credentials(
					credential_created_at DESC,
					credential_id DESC);`,
		DownSQL: `DROP TABLE credentials;`,
	},
	{
		SequenceNumber: 17,
		Name:           "create_artifacts",
		UpSQL: `CREATE TABLE IF NOT EXISTS artifacts
				(
					artifact_id text NOT NULL PRIMARY KEY,
					artifact_name text NOT NULL,
					artifact_step_id text NOT NULL REFERENCES steps (step_id) ON UPDATE NO ACTION ON DELETE NO ACTION,
					artifact_created_at timestamp without time zone NOT NULL,
					artifact_updated_at timestamp without time zone NOT NULL,
					artifact_group_name text NOT NULL,
					artifact_etag text NOT NULL,
					artifact_size integer NOT NULL,
					artifact_mime text,
					artifact_path text NOT NULL,
					artifact_hash_type text,
					artifact_hash text,
					artifact_sealed bool NOT NULL
				);
				CREATE UNIQUE INDEX IF NOT EXISTS artifacts_name_unique_index ON artifacts(
					artifact_step_id,
					artifact_name);
				CREATE UNIQUE INDEX IF NOT EXISTS artifacts_path_unique_index ON artifacts(
					artifact_step_id,
					artifact_path);
				CREATE UNIQUE INDEX IF NOT EXISTS artifacts_created_at_id_desc_unique_index ON artifacts(
					artifact_created_at DESC,
					artifact_id DESC);`,
		DownSQL: `DROP TABLE artifacts;`,
	},
	{
		SequenceNumber: 18,
		Name:           "create_legal_entities_memberships",
		UpSQL: `CREATE TABLE IF NOT EXISTS legal_entities_memberships
				(
					legal_entities_membership_id text NOT NULL PRIMARY KEY,
					legal_entities_membership_created_at timestamp without time zone NOT NULL,
					legal_entities_membership_legal_entity_id text NOT NULL REFERENCES legal_entities (legal_entity_id) ON UPDATE NO ACTION ON DELETE NO ACTION,
					legal_entities_membership_member_legal_entity_id text NOT NULL REFERENCES legal_entities (legal_entity_id) ON UPDATE NO ACTION ON DELETE NO ACTION
				);
				CREATE UNIQUE INDEX IF NOT EXISTS legal_entities_memberships_unique_index ON legal_entities_memberships(
					legal_entities_membership_legal_entity_id,
					legal_entities_membership_member_legal_entity_id);
				CREATE UNIQUE INDEX IF NOT EXISTS legal_entities_memberships_created_at_id_desc_unique_index ON legal_entities_memberships(
					legal_entities_membership_created_at DESC,
					legal_entities_membership_id DESC);`,
		DownSQL: `DROP TABLE legal_entities_memberships;`,
	},
	{
		SequenceNumber: 19,
		Name:           "create_repo_build_counters",
		UpSQL: `CREATE TABLE IF NOT EXISTS repo_build_counters
				(
					repo_build_counter_repo_id text NOT NULL REFERENCES repos (repo_id) ON UPDATE NO ACTION ON DELETE NO ACTION PRIMARY KEY,
					repo_build_counter_counter integer NOT NULL DEFAULT 0
				);
				CREATE UNIQUE INDEX IF NOT EXISTS repo_build_counters_repo_id_unique_index ON repo_build_counters(repo_build_counter_repo_id);`,
		DownSQL: `DROP TABLE repo_build_counters;`,
	},
	{
		SequenceNumber: 20,
		Name:           "create_resource_link_fragments",
		UpSQL: `CREATE TABLE IF NOT EXISTS resource_link_fragments
				(
					resource_link_fragment_id text NOT NULL PRIMARY KEY,
					resource_link_fragment_name text NOT NULL,
					resource_link_fragment_parent_id text,
					resource_link_fragment_kind text NOT NULL,
					resource_link_fragment_created_at timestamp without time zone NOT NULL
				);
				CREATE UNIQUE INDEX IF NOT EXISTS resource_link_fragment_name_unique_index ON resource_link_fragments(
					resource_link_fragment_parent_id,
					resource_link_fragment_kind,
					resource_link_fragment_name);
				CREATE UNIQUE INDEX IF NOT EXISTS resource_link_fragments_created_at_id_desc_unique_index ON resource_link_fragments(
					resource_link_fragment_created_at DESC,
					resource_link_fragment_id DESC);`,
		DownSQL: `DROP TABLE resource_link_fragments;`,
	},
	{
		SequenceNumber: 21,
		Name:           "create_pull_requests",
		UpSQL: `CREATE TABLE IF NOT EXISTS pull_requests
				(
					pull_request_id text NOT NULL PRIMARY KEY,
					pull_request_created_at timestamp without time zone NOT NULL,
					pull_request_updated_at timestamp without time zone NOT NULL,
					pull_request_merged_at timestamp without time zone,
					pull_request_closed_at timestamp without time zone,
					pull_request_title text,
					pull_request_state text,
					pull_request_repo_id text NOT NULL REFERENCES repos (repo_id) ON UPDATE NO ACTION ON DELETE NO ACTION,
					pull_request_user_id text REFERENCES legal_entities (legal_entity_id) ON UPDATE NO ACTION ON DELETE NO ACTION,
					pull_request_base_ref text NOT NULL,
					pull_request_head_ref text NOT NULL,
					pull_request_external_id text
				);
				CREATE UNIQUE INDEX IF NOT EXISTS pull_requests_external_id_unique_index ON pull_requests(pull_request_external_id);
				CREATE UNIQUE INDEX IF NOT EXISTS pull_requests_created_at_id_desc_unique_index ON pull_requests(
					pull_request_created_at DESC,
					pull_request_id DESC);`,
		DownSQL: `DROP TABLE pull_requests;`,
	},
	{
		SequenceNumber: 22,
		Name:           "create_work_item_states",
		UpSQL: `CREATE TABLE IF NOT EXISTS work_item_states
				(
					work_item_state_id text NOT NULL PRIMARY KEY,
					work_item_state_attempts_so_far INTEGER NOT NULL,
					work_item_state_not_before timestamp without time zone,
					work_item_state_allocated_at timestamp without time zone,
					work_item_state_allocated_to text,
					work_item_state_allocated_until timestamp without time zone,
					work_item_state_created_at timestamp without time zone NOT NULL
				);
				CREATE UNIQUE INDEX IF NOT EXISTS work_item_states_created_at_id_desc_unique_index ON work_item_states(
					work_item_state_created_at DESC,
					work_item_state_id DESC);`,
		DownSQL: `DROP TABLE work_item_states;`,
	},
	{
		SequenceNumber: 23,
		Name:           "create_work_items",
		UpSQL: `CREATE TABLE IF NOT EXISTS work_items
				(
					work_item_id text NOT NULL PRIMARY KEY,
					work_item_created_at timestamp without time zone NOT NULL,
					work_item_concurrency_key text,
					work_item_state text NOT NULL REFERENCES work_item_states (work_item_state_id) ON UPDATE NO ACTION ON DELETE NO ACTION,
					work_item_type text NOT NULL,
					work_item_data text NOT NULL,
					work_item_status text NOT NULL,
					work_item_completed_at timestamp without time zone
				);
				CREATE UNIQUE INDEX IF NOT EXISTS work_items_created_at_id_desc_unique_index ON work_items(
					work_item_created_at DESC,
					work_item_id DESC);`,
		DownSQL: `DROP TABLE work_items;`,
	},
	{
		SequenceNumber: 24,
		Name:           "create_credentials_indices",
		UpSQL: `CREATE UNIQUE INDEX IF NOT EXISTS credentials_shared_secret_id_unique_index ON credentials(credential_shared_secret_id)
					WHERE credential_shared_secret_id IS NOT NULL AND credential_shared_secret_id != '';
				CREATE UNIQUE INDEX IF NOT EXISTS credentials_github_user_id_unique_index ON credentials(credential_github_user_id)
					WHERE credential_github_user_id IS NOT NULL AND credential_github_user_id != '' AND credential_github_user_id != '0';
				CREATE UNIQUE INDEX IF NOT EXISTS credentials_client_public_key_asn1_hash_unique_index ON credentials(credential_client_public_key_asn1_hash)
					WHERE credential_client_public_key_asn1_hash IS NOT NULL AND credential_client_public_key_asn1_hash != '';`,
		DownSQL: `DROP INDEX credentials_shared_secret_id_unique_index;
				  DROP INDEX credentials_github_user_id_unique_index;
				  DROP INDEX credentials_client_public_key_asn1_hash_unique_index;`,
	},
	{
		SequenceNumber: 25,
		Name:           "alter_credentials_github_user_id",
		UpSQL: `DROP INDEX credentials_github_user_id_unique_index;
                ALTER TABLE credentials DROP COLUMN credential_github_user_id;
				ALTER TABLE credentials ADD credential_github_user_id integer;
                CREATE UNIQUE INDEX credentials_github_user_id_unique_index ON credentials(credential_github_user_id)
					WHERE credential_github_user_id IS NOT NULL AND credential_github_user_id != 0;`,
		DownSQL: `DROP INDEX credentials_github_user_id_unique_index;
				  ALTER TABLE credentials DROP COLUMN credential_github_user_id;
				  ALTER TABLE credentials ADD credential_github_user_id text;
				  CREATE UNIQUE INDEX IF NOT EXISTS credentials_github_user_id_unique_index ON credentials(credential_github_user_id)
					WHERE credential_github_user_id IS NOT NULL AND credential_github_user_id != '' AND credential_github_user_id != '0';`,
	},
	{
		SequenceNumber: 26,
		Name:           "move_docker_config_from_step_to_job",
		UpSQL: `ALTER TABLE steps DROP COLUMN step_docker_image;
				ALTER TABLE steps DROP COLUMN step_docker_image_pull_strategy;
				ALTER TABLE steps DROP COLUMN step_docker_authentication;
				ALTER TABLE jobs ADD COLUMN job_docker_image text;
				ALTER TABLE jobs ADD COLUMN job_docker_image_pull_strategy text;
				ALTER TABLE jobs ADD COLUMN job_docker_auth text;`,
		DownSQL: `ALTER TABLE jobs DROP COLUMN job_docker_image;
				ALTER TABLE jobs DROP COLUMN job_docker_image_pull_strategy;
				ALTER TABLE jobs DROP COLUMN job_docker_auth;
				ALTER TABLE steps ADD COLUMN step_docker_image text NOT NULL;
				ALTER TABLE steps ADD COLUMN step_docker_image_pull_strategy text NOT NULL;
				ALTER TABLE steps ADD COLUMN step_docker_authentication text;`,
	},
	{
		SequenceNumber: 27,
		Name:           "move_fingerprint_from_step_to_job",
		UpSQL: `ALTER TABLE steps DROP COLUMN step_indirect_to_step_id;
				ALTER TABLE steps DROP COLUMN step_fingerprint_commands;
				ALTER TABLE steps DROP COLUMN step_fingerprint;
				ALTER TABLE steps DROP COLUMN step_fingerprint_hash_type;
				ALTER TABLE jobs ADD COLUMN job_indirect_to_job_id text REFERENCES jobs (job_id) ON UPDATE NO ACTION ON DELETE NO ACTION;
				ALTER TABLE jobs ADD COLUMN job_fingerprint_commands text;
				ALTER TABLE jobs ADD COLUMN job_fingerprint text;
				ALTER TABLE jobs ADD COLUMN job_fingerprint_hash_type text;`,
		DownSQL: `ALTER TABLE jobs DROP COLUMN job_indirect_to_job_id;
				ALTER TABLE jobs DROP COLUMN job_fingerprint_commands;
				ALTER TABLE jobs DROP COLUMN job_fingerprint;
				ALTER TABLE jobs DROP COLUMN job_fingerprint_hash_type;
				ALTER TABLE steps ADD COLUMN step_indirect_to_step_id text REFERENCES steps (step_id) ON UPDATE NO ACTION ON DELETE NO ACTION;
				ALTER TABLE steps ADD COLUMN step_fingerprint_commands text;
				ALTER TABLE steps ADD COLUMN step_fingerprint text;
				ALTER TABLE steps ADD COLUMN step_fingerprint_hash_type text;`,
	},
	{
		SequenceNumber: 28,
		Name:           "add_repo_ssh_key_secret_id",
		UpSQL:          `ALTER TABLE repos ADD COLUMN repo_ssh_key_secret_id text REFERENCES secrets (secret_id) ON UPDATE NO ACTION ON DELETE NO ACTION;`,
		DownSQL:        `ALTER TABLE repos DROP COLUMN repo_ssh_key_secret_id;`,
	},
	{
		SequenceNumber: 29,
		Name:           "create_repos_foreign_key_indexes",
		UpSQL:          `CREATE INDEX IF NOT EXISTS repos_repo_legal_entity_id_index ON repos(repo_legal_entity_id);`,
		DownSQL:        `DROP INDEX repos_repo_legal_entity_id_index;`,
	},
	{
		SequenceNumber: 30,
		Name:           "create_commits_foreign_key_indexes",
		UpSQL: `CREATE INDEX IF NOT EXISTS commits_commit_author_id_index ON commits(commit_author_id);
				CREATE INDEX IF NOT EXISTS commits_commit_committer_id_index ON commits(commit_committer_id);`,
		DownSQL: `DROP INDEX commits_commit_author_id_index; 
				  DROP INDEX commits_commit_committer_id_index;`,
	},
	{
		SequenceNumber: 31,
		Name:           "create_builds_foreign_key_indexes",
		UpSQL:          `CREATE INDEX IF NOT EXISTS builds_build_commit_id_index ON builds(build_commit_id);`,
		DownSQL:        `DROP INDEX builds_build_commit_id_index;`,
	},
	{
		SequenceNumber: 32,
		Name:           "create_jobs_foreign_key_indexes",
		UpSQL: `CREATE INDEX IF NOT EXISTS jobs_job_repo_id_index ON jobs(job_repo_id);
				CREATE INDEX IF NOT EXISTS jobs_job_runner_id_index ON jobs(job_runner_id);`,
		DownSQL: `DROP INDEX jobs_job_repo_id_index;
				  DROP INDEX jobs_job_runner_id_index;`,
	},
	{
		SequenceNumber: 33,
		Name:           "create_jobs_depend_on_jobs_foreign_key_indexes",
		UpSQL: `CREATE INDEX IF NOT EXISTS jobs_depend_on_jobs_source_job_id_index ON jobs_depend_on_jobs(jobs_depend_on_jobs_source_job_id); 
				CREATE INDEX IF NOT EXISTS jobs_depend_on_jobs_target_job_id_index ON jobs_depend_on_jobs(jobs_depend_on_jobs_target_job_id);`,
		DownSQL: `DROP INDEX jobs_depend_on_jobs_source_job_id_index; 
				  DROP INDEX jobs_depend_on_jobs_target_job_id_index; `,
	},
	{
		SequenceNumber: 34,
		Name:           "create_steps_foreign_key_indexes",
		UpSQL:          `CREATE INDEX IF NOT EXISTS steps_step_repo_id_index ON steps(step_repo_id);`,
		DownSQL:        `DROP INDEX steps_step_repo_id_index;`,
	},
	{
		SequenceNumber: 35,
		Name:           "create_secrets_foreign_key_indexes",
		UpSQL:          `CREATE INDEX IF NOT EXISTS secrets_secret_repo_id_id_index ON secrets(secret_repo_id);`,
		DownSQL:        `DROP INDEX secrets_secret_repo_id_id_index;`,
	},
	{
		SequenceNumber: 36,
		Name:           "create_access_control_groups_foreign_key_indexes",
		UpSQL:          `CREATE INDEX IF NOT EXISTS access_control_groups_legal_entity_id_index ON access_control_groups(access_control_group_legal_entity_id);`,
		DownSQL:        `DROP INDEX access_control_groups_legal_entity_id_index;`,
	},
	{
		SequenceNumber: 37,
		Name:           "create_access_control_group_memberships_foreign_key_indexes",
		UpSQL: `CREATE INDEX IF NOT EXISTS access_control_group_membership_added_by_legal_entity_id_index ON access_control_group_memberships(access_control_group_membership_added_by_legal_entity_id); 
				CREATE INDEX IF NOT EXISTS access_control_group_membership_group_id_index ON access_control_group_memberships(access_control_group_membership_group_id);
				CREATE INDEX IF NOT EXISTS access_control_group_membership_member_identity_id_index ON access_control_group_memberships(access_control_group_membership_member_identity_id);`,
		DownSQL: `DROP INDEX access_control_group_membership_added_by_legal_entity_id_index; 
				  DROP INDEX access_control_group_membership_group_id_index; 
				  DROP INDEX access_control_group_membership_member_identity_id_index; `,
	},
	{
		SequenceNumber: 38,
		Name:           "create_access_control_grants_foreign_key_indexes",
		UpSQL: `CREATE INDEX IF NOT EXISTS access_control_grant_granted_by_legal_entity_id_index ON access_control_grants(access_control_grant_granted_by_legal_entity_id); 
				CREATE INDEX IF NOT EXISTS access_control_grant_authorized_identity_id_index ON access_control_grants(access_control_grant_authorized_identity_id);
				CREATE INDEX IF NOT EXISTS access_control_grant_authorized_group_id_index ON access_control_grants(access_control_grant_authorized_group_id);
				CREATE INDEX IF NOT EXISTS access_control_grant_target_resource_id_index ON access_control_grants(access_control_grant_target_resource_id);`,
		DownSQL: `DROP INDEX access_control_grant_granted_by_legal_entity_id_index; 
				  DROP INDEX access_control_grant_authorized_identity_id_index; 
				  DROP INDEX access_control_grant_authorized_group_id_index; 
				  DROP INDEX access_control_grant_target_resource_id_index; `,
	},
	{
		SequenceNumber: 39,
		Name:           "create_credentials_indexes",
		UpSQL:          `CREATE INDEX IF NOT EXISTS credentials_credential_identity_id_index ON credentials(credential_identity_id);`,
		DownSQL:        `DROP INDEX credentials_credential_identity_id_index;`,
	},
	{
		SequenceNumber: 40,
		Name:           "create_legal_entities_memberships_indexes",
		UpSQL:          `CREATE INDEX IF NOT EXISTS legal_entities_membership_member_legal_entity_id_index ON legal_entities_memberships(legal_entities_membership_member_legal_entity_id);`,
		DownSQL:        `DROP INDEX legal_entities_membership_member_legal_entity_id_index;`,
	},
	{
		SequenceNumber: 41,
		Name:           "create_pull_requests_indexes",
		UpSQL: `CREATE INDEX IF NOT EXISTS pull_requests_pull_request_repo_id_index ON pull_requests(pull_request_repo_id); 
				CREATE INDEX IF NOT EXISTS pull_requests_pull_request_user_id_index ON pull_requests(pull_request_user_id);`,
		DownSQL: `DROP INDEX pull_requests_pull_request_repo_id_index;
				  DROP INDEX pull_requests_pull_request_user_id_index;`,
	},
	{
		SequenceNumber: 42,
		Name:           "create_work_items_indexes",
		UpSQL:          `CREATE INDEX IF NOT EXISTS work_items_work_item_state_index ON work_items(work_item_state);`,
		DownSQL:        `DROP INDEX work_items_work_item_state_index;`,
	},
	{
		SequenceNumber: 43,
		Name:           "create_access_control_ownerships_indexes",
		UpSQL:          `CREATE INDEX IF NOT EXISTS access_control_ownership_owner_resource_id_index ON access_control_ownerships(access_control_ownership_owner_resource_id);`,
		DownSQL:        `DROP INDEX access_control_ownership_owner_resource_id_index;`,
	},
	{
		SequenceNumber: 44,
		Name:           "drop_repos_build_counter_column",
		// this is a not used as the counter is in repos_build_counters table
		UpSQL:   `ALTER TABLE repos DROP COLUMN repo_build_counter;`,
		DownSQL: `ALTER TABLE repos ADD COLUMN repo_build_counter integer NOT NULL DEFAULT 0;`,
	},
	{
		SequenceNumber: 45,
		Name:           "create work_item_state_indexes",
		UpSQL: `CREATE INDEX IF NOT EXISTS work_item_state_allocated_until_index ON work_item_states(work_item_state_allocated_until)
				  		WHERE work_item_state_allocated_until IS NOT NULL;
			    CREATE INDEX IF NOT EXISTS work_item_state_not_before_index ON work_item_states(work_item_state_not_before)
				  		WHERE work_item_state_not_before IS NOT NULL;`,
		DownSQL: `DROP INDEX work_item_state_allocated_until_index;
				  DROP INDEX work_item_state_not_before_index;`,
	},
	{
		SequenceNumber: 46,
		Name:           "add_runner_runtime_fields",
		UpSQL: `ALTER TABLE runners ADD COLUMN runner_operating_system text NOT NULL DEFAULT '';
                ALTER TABLE runners ADD COLUMN runner_architecture text NOT NULL DEFAULT '';`,
		DownSQL: `ALTER TABLE runners DROP COLUMN runner_operating_system;
                  ALTER TABLE runners DROP COLUMN runner_architecture;`,
	},
	{
		SequenceNumber: 47,
		Name:           "add_group_external_id",
		UpSQL: `ALTER TABLE access_control_groups ADD COLUMN access_control_group_external_id text DEFAULT NULL;
                CREATE INDEX IF NOT EXISTS access_control_groups_external_id_unique_index ON access_control_groups(access_control_group_external_id)
				    WHERE access_control_group_deleted_at IS NULL;`,
		DownSQL: `DROP INDEX access_control_groups_external_id_unique_index;
                  ALTER TABLE access_control_groups DROP COLUMN access_control_group_external_id;`,
	},
	{
		SequenceNumber: 48,
		Name:           "add_job_docker_shell",
		UpSQL:          `ALTER TABLE jobs ADD COLUMN job_docker_shell text;`,
		DownSQL:        `ALTER TABLE jobs DROP COLUMN job_docker_shell;`,
	},
	{
		SequenceNumber: 49,
		Name:           "add_job_definition_hash",
		UpSQL: `ALTER TABLE jobs ADD COLUMN job_definition_hash text NOT NULL DEFAULT 'unknown';
	            ALTER TABLE jobs ADD COLUMN job_definition_hash_type text NOT NULL DEFAULT 'unknown';`,
		DownSQL: `ALTER TABLE jobs DROP COLUMN job_definition_hash;
				  ALTER TABLE jobs DROP COLUMN job_definition_hash_type;`,
	},
	{
		SequenceNumber: 50,
		Name:           "remove_job_definition_hash",
		UpSQL: `ALTER TABLE jobs DROP COLUMN job_definition_hash;
				  ALTER TABLE jobs DROP COLUMN job_definition_hash_type;`,
		DownSQL: `ALTER TABLE jobs ADD COLUMN job_definition_hash text NOT NULL DEFAULT 'unknown';
	            ALTER TABLE jobs ADD COLUMN job_definition_hash_type text NOT NULL DEFAULT 'unknown';`,
	},
	{
		SequenceNumber: 51,
		Name:           "add_group_membership_external_id",
		UpSQL: `ALTER TABLE access_control_group_memberships ADD COLUMN access_control_group_membership_source_system text DEFAULT '';
	       	DROP INDEX group_membership_group_id_identity_id_unique_index;
				CREATE UNIQUE INDEX group_membership_members_unique_index ON access_control_group_memberships(
					access_control_group_membership_group_id,
					access_control_group_membership_member_identity_id,
				    access_control_group_membership_source_system);`,
		DownSQL: `DROP INDEX group_membership_members_unique_index;
	           ALTER TABLE access_control_group_memberships DROP COLUMN access_control_group_membership_source_system;
				CREATE UNIQUE INDEX group_membership_group_id_identity_id_unique_index ON access_control_group_memberships(
					access_control_group_membership_group_id,
					access_control_group_membership_member_identity_id);`,
	},
	{
		SequenceNumber: 52,
		Name:           "add_job_runs_on",
		UpSQL:          `ALTER TABLE jobs ADD COLUMN job_runs_on text;`,
		DownSQL:        `ALTER TABLE jobs DROP COLUMN job_runs_on;`,
	},
	{
		SequenceNumber: 53,
		Name:           "create_job_labels",
		UpSQL: `CREATE TABLE IF NOT EXISTS job_labels
				(
					job_label_job_id text REFERENCES jobs (job_id) ON UPDATE NO ACTION ON DELETE CASCADE,
					job_label_label text NOT NULL
				);
				CREATE UNIQUE INDEX IF NOT EXISTS job_labels_unique ON job_labels(
					job_label_job_id,
					job_label_label);`,
		DownSQL: `DROP INDEX job_labels_unique;
				  DROP TABLE job_labels;`,
	},
	{
		SequenceNumber: 54,
		Name:           "add_runner_labels",
		UpSQL:          `ALTER TABLE runners ADD COLUMN runner_labels text;`,
		DownSQL:        `ALTER TABLE runners DROP COLUMN runner_labels;`,
	},
	{
		SequenceNumber: 55,
		Name:           "create_runner_labels",
		UpSQL: `CREATE TABLE IF NOT EXISTS runner_labels
				(
					runner_label_runner_id text REFERENCES runners (runner_id) ON UPDATE NO ACTION ON DELETE CASCADE,
					runner_label_label text NOT NULL
				);
				CREATE UNIQUE INDEX IF NOT EXISTS runner_labels_unique ON runner_labels(
					runner_label_runner_id,
					runner_label_label);`,
		DownSQL: `DROP INDEX runner_labels_unique;
				  DROP TABLE runner_labels;`,
	},
	{
		SequenceNumber: 56,
		Name:           "add_runner_supported_job_types",
		UpSQL:          `ALTER TABLE runners ADD COLUMN runner_supported_job_types text;`,
		DownSQL:        `ALTER TABLE runners DROP COLUMN runner_supported_job_types;`,
	},
	{
		SequenceNumber: 57,
		Name:           "create_runner_supported_job_types",
		UpSQL: `CREATE TABLE IF NOT EXISTS runner_supported_job_types
				(
					runner_supported_job_types_runner_id text REFERENCES runners (runner_id) ON UPDATE NO ACTION ON DELETE CASCADE,
					runner_supported_job_types_job_type text NOT NULL
				);
				CREATE UNIQUE INDEX IF NOT EXISTS runner_supported_job_types_unique ON runner_supported_job_types(
					runner_supported_job_types_runner_id,
					runner_supported_job_types_job_type);`,
		DownSQL: `DROP INDEX runner_supported_job_types_unique;
				  DROP TABLE runner_supported_job_types;`,
	},
	{
		SequenceNumber: 58,
		Name:           "create_events",
		UpSQL: `CREATE TABLE IF NOT EXISTS events
				(
					event_id text NOT NULL PRIMARY KEY,
					event_created_at timestamp without time zone NOT NULL,
					event_build_id text NOT NULL,
					event_sequence_number integer NOT NULL,
					event_type text NOT NULL,
					event_resource_id text NOT NULL,
					event_resource_name text NOT NULL,
					event_payload text
				);
				CREATE UNIQUE INDEX IF NOT EXISTS events_sequence_number_unique_index ON events(
					event_build_id,
					event_sequence_number);`,
		DownSQL: `DROP TABLE events;`,
	},
	{
		SequenceNumber: 59,
		Name:           "create_build_event_counters",
		UpSQL: `CREATE TABLE IF NOT EXISTS build_event_counters
				(
					build_event_counter_build_id text NOT NULL REFERENCES builds (build_id) ON UPDATE NO ACTION ON DELETE NO ACTION PRIMARY KEY,
					build_event_counter_counter integer NOT NULL DEFAULT 0
				);
				CREATE UNIQUE INDEX IF NOT EXISTS build_event_counters_build_id_unique_index ON build_event_counters(build_event_counter_build_id);`,
		DownSQL: `DROP TABLE build_event_counters;`,
	},
	{
		SequenceNumber: 60,
		Name:           "add_legal_entity_synced_at",
		UpSQL:          `ALTER TABLE legal_entities ADD COLUMN legal_entity_synced_at timestamp without time zone;`,
		DownSQL:        `ALTER TABLE legal_entities DROP COLUMN legal_entity_synced_at;`,
	},
	{
		SequenceNumber: 61,
		Name:           "move_artifacts_from_step_to_job",
		UpSQL: `ALTER TABLE jobs ADD COLUMN job_artifact_definitions text;
                ALTER TABLE steps DROP COLUMN step_artifact_definitions;
				DROP INDEX artifacts_name_unique_index;
				DROP INDEX artifacts_path_unique_index;
			    ALTER TABLE artifacts ADD COLUMN artifact_job_id text REFERENCES jobs (job_id) ON UPDATE NO ACTION ON DELETE NO ACTION;
				ALTER TABLE artifacts DROP COLUMN artifact_step_id;
				CREATE UNIQUE INDEX IF NOT EXISTS artifacts_name_unique_index ON artifacts(
					artifact_job_id,
					artifact_name);
				CREATE UNIQUE INDEX IF NOT EXISTS artifacts_path_unique_index ON artifacts(
					artifact_job_id,
					artifact_path);`,
		DownSQL: `ALTER TABLE jobs DROP COLUMN job_artifact_definitions;
                ALTER TABLE steps ADD COLUMN step_artifact_definitions text;
                DROP INDEX artifacts_name_unique_index;
				DROP INDEX artifacts_path_unique_index;
			    ALTER TABLE artifacts DROP COLUMN artifact_job_id;
				ALTER TABLE artifacts ADD COLUMN artifact_step_id text REFERENCES steps (step_id) ON UPDATE NO ACTION ON DELETE NO ACTION;
				CREATE UNIQUE INDEX IF NOT EXISTS artifacts_name_unique_index ON artifacts(
					artifact_step_id,
					artifact_name);
				CREATE UNIQUE INDEX IF NOT EXISTS artifacts_path_unique_index ON artifacts(
					artifact_step_id,
					artifact_path);`,
	},
	{
		SequenceNumber: 62,
		Name:           "add_job_definition_data_hashes",
		UpSQL: `ALTER TABLE jobs ADD COLUMN job_definition_data_hash_type text DEFAULT '';
				ALTER TABLE jobs ADD COLUMN job_definition_data_hash text DEFAULT '';`,
		DownSQL: `ALTER TABLE jobs DROP COLUMN job_definition_data_hash_type;
				  ALTER TABLE jobs DROP COLUMN job_definition_data_hash;`,
	},
	{
		SequenceNumber: 63,
		Name:           "move_environment_from_step_to_job",
		UpSQL: `ALTER TABLE jobs ADD COLUMN job_environment text;
                ALTER TABLE steps DROP COLUMN step_environment;`,
		DownSQL: `ALTER TABLE jobs DROP COLUMN job_environment;
                ALTER TABLE steps ADD COLUMN step_environment text;`,
	},
	{
		SequenceNumber: 64,
		Name:           "add_runner_enabled",
		UpSQL:          `ALTER TABLE runners ADD COLUMN runner_enabled bool NOT NULL default TRUE;`,
		DownSQL:        `ALTER TABLE runners DROP COLUMN runner_enabled;`,
	},
	{
		SequenceNumber: 65,
		Name:           "job_workflows",
		UpSQL: `ALTER TABLE jobs ADD COLUMN job_workflow text NOT NULL DEFAULT '';
				DROP INDEX jobs_job_name_unique_index;
				CREATE UNIQUE INDEX IF NOT EXISTS jobs_job_name_unique_index ON jobs(
					job_build_id,
				    job_workflow,
					job_name);
                ALTER TABLE jobs_depend_on_jobs ADD COLUMN jobs_depend_on_jobs_build_id text REFERENCES builds(build_id) ON UPDATE NO ACTION ON DELETE NO ACTION;
                ALTER TABLE jobs_depend_on_jobs ADD COLUMN jobs_depend_on_jobs_target_workflow text;
                ALTER TABLE jobs_depend_on_jobs ADD COLUMN jobs_depend_on_jobs_target_job_name text;
				DROP INDEX jobs_depend_on_jobs_target_job_id_index;
				ALTER TABLE jobs_depend_on_jobs DROP COLUMN jobs_depend_on_jobs_target_job_id;
				ALTER TABLE jobs_depend_on_jobs ADD COLUMN jobs_depend_on_jobs_target_job_id text REFERENCES jobs (job_id) ON UPDATE NO ACTION ON DELETE NO ACTION;
                CREATE INDEX jobs_depend_on_jobs_target_job_id_index ON jobs_depend_on_jobs(jobs_depend_on_jobs_target_job_id);`,
		DownSQL: `DROP INDEX jobs_job_name_unique_index;
                  CREATE UNIQUE INDEX IF NOT EXISTS jobs_job_name_unique_index ON jobs(
					job_build_id,
					job_name);
                  ALTER TABLE jobs DROP COLUMN job_workflow;
                  ALTER TABLE jobs_depend_on_jobs DROP COLUMN jobs_depend_on_jobs_build_id;
                  ALTER TABLE jobs_depend_on_jobs DROP COLUMN jobs_depend_on_jobs_target_workflow;
                  ALTER TABLE jobs_depend_on_jobs DROP COLUMN jobs_depend_on_jobs_target_job_name;
                  DROP INDEX jobs_depend_on_jobs_target_job_id_index;
				  ALTER TABLE jobs_depend_on_jobs DROP COLUMN jobs_depend_on_jobs_target_job_id;
				  ALTER TABLE jobs_depend_on_jobs ADD COLUMN jobs_depend_on_jobs_target_job_id text NOT NULL REFERENCES jobs (job_id) ON UPDATE NO ACTION ON DELETE NO ACTION;
                  CREATE INDEX jobs_depend_on_jobs_target_job_id_index ON jobs_depend_on_jobs(jobs_depend_on_jobs_target_job_id);`,
	},
	{
		SequenceNumber: 66,
		Name:           "event_workflows",
		UpSQL: `ALTER TABLE events ADD COLUMN event_workflow text NOT NULL DEFAULT '';
				ALTER TABLE events ADD COLUMN event_job_name text NOT NULL DEFAULT '';`,
		DownSQL: `ALTER TABLE events DROP COLUMN event_workflow;
                  ALTER TABLE events DROP COLUMN event_job_name;`,
	},
	{
		SequenceNumber: 67,
		Name:           "create_jobs_depend_on_jobs_workflows_index",
		UpSQL: `CREATE INDEX jobs_depend_on_jobs_target_job_name_index ON jobs_depend_on_jobs(
					jobs_depend_on_jobs_build_id,
					jobs_depend_on_jobs_target_workflow,
					jobs_depend_on_jobs_target_job_name,
					jobs_depend_on_jobs_target_job_id);`,
		DownSQL: `DROP INDEX jobs_depend_on_jobs_target_job_name_index; `,
	},
}
