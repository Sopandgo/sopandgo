package audit

var strictAuditTypes = false

const (
	EntitySystem         = "system"
	EntityUser           = "user"
	EntitySOP            = "sop"
	EntitySOPVersion     = "sop_version"
	EntitySOPAsset       = "sop_asset"
	EntityTag            = "tag"
	EntityEmail          = "email"
	EntityAcknowledgment = "acknowledgment"
	EntityNotification   = "notification"
	EntityIntegration    = "integration"
)

const (
	EventSystemRestart                = "system_restart"
	EventIntegrityCheck               = "integrity_check"
	EventLogin                        = "login"
	EventLoginFailed                  = "login_failed"
	EventLogout                       = "logout"
	EventUserCreated                  = "user_created"
	EventUserRoleUpdated              = "user_role_updated"
	EventUserStatusChanged            = "user_status_changed"
	EventUserPasswordUpdated          = "user_password_updated"
	EventAdminRevokedAllUserSessions  = "admin_revoked_all_user_sessions"
	EventSystemGlobalRevocation       = "system_global_revocation"
	EventSOPCreated                   = "sop_created"
	EventSOPVersionCreated            = "sop_version_created"
	EventSOPVersionStateChanged       = "sop_version_state_changed"
	EventSOPAcknowledgmentCreated     = "sop_acknowledgment_created"
	EventAcknowledgmentAdded          = "acknowledgment_added"
	EventAssetAdded                   = "asset_added"
	EventTagCreated                   = "tag_created"
	EventTagAttached                  = "tag_attached"
	EventTagDetached                  = "tag_detached"
	EventTagRetired                   = "tag_retired"
	EventTagRevived                   = "tag_revived"
	EventEmailSent                    = "email_sent"
	EventEmailFailed                  = "email_failed"
	EventSmtpSettingsUpdated          = "smtp_settings_updated"
	EventMailModeUpdated              = "mail_mode_updated"
	EventDefaultLocaleUpdated         = "default_locale_updated"
	EventMailTransportUpdated         = "mail_transport_updated"
	EventResendSettingsUpdated        = "resend_settings_updated"
	EventManualInviteLinkGenerated    = "manual_invite_link_generated"
	EventManualResetLinkGenerated     = "manual_reset_link_generated"
	EventBackupExported               = "backup_exported"
	EventBackupImportValidated        = "backup_import_validated"
	EventBackupApplyStaged            = "backup_apply_staged"
	EventBackupS3Uploaded             = "backup_s3_uploaded"
	EventBackupS3Failed               = "backup_s3_failed"
	EventBackupS3SettingsUpdated      = "backup_s3_settings_updated"
	EventSOPVersionPDFArtifactCreated = "sop_version_pdf_artifact_created"
	EventSOPFavoriteAdded             = "sop_favorite_added"
	EventSOPFavoriteRemoved           = "sop_favorite_removed"
	EventNotificationSent             = "notification_sent"
	EventNotificationFailed           = "notification_failed"
	EventIntegrationSettingsUpdated   = "integration_settings_updated"
)

var knownEventTypes = map[string]struct{}{
	EventSystemRestart:                {},
	EventIntegrityCheck:               {},
	EventLogin:                        {},
	EventLoginFailed:                  {},
	EventLogout:                       {},
	EventUserCreated:                  {},
	EventUserRoleUpdated:              {},
	EventUserStatusChanged:            {},
	EventUserPasswordUpdated:          {},
	EventAdminRevokedAllUserSessions:  {},
	EventSystemGlobalRevocation:       {},
	EventSOPCreated:                   {},
	EventSOPVersionCreated:            {},
	EventSOPVersionStateChanged:       {},
	EventSOPAcknowledgmentCreated:     {},
	EventAcknowledgmentAdded:          {},
	EventAssetAdded:                   {},
	EventTagCreated:                   {},
	EventTagAttached:                  {},
	EventTagDetached:                  {},
	EventTagRetired:                   {},
	EventTagRevived:                   {},
	EventEmailSent:                    {},
	EventEmailFailed:                  {},
	EventSmtpSettingsUpdated:          {},
	EventMailModeUpdated:              {},
	EventDefaultLocaleUpdated:         {},
	EventMailTransportUpdated:         {},
	EventResendSettingsUpdated:        {},
	EventManualInviteLinkGenerated:    {},
	EventManualResetLinkGenerated:     {},
	EventBackupExported:               {},
	EventBackupImportValidated:        {},
	EventBackupApplyStaged:            {},
	EventBackupS3Uploaded:             {},
	EventBackupS3Failed:               {},
	EventBackupS3SettingsUpdated:      {},
	EventSOPVersionPDFArtifactCreated: {},
	EventSOPFavoriteAdded:             {},
	EventSOPFavoriteRemoved:           {},
	EventNotificationSent:             {},
	EventNotificationFailed:           {},
	EventIntegrationSettingsUpdated:   {},
}

var knownEntityTypes = map[string]struct{}{
	EntitySystem:         {},
	EntityUser:           {},
	EntitySOP:            {},
	EntitySOPVersion:     {},
	EntitySOPAsset:       {},
	EntityTag:            {},
	EntityEmail:          {},
	EntityAcknowledgment: {},
	EntityNotification:   {},
	EntityIntegration:    {},
}
