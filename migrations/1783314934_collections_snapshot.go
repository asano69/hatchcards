package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		jsonData := `[
			{
				"createRule": null,
				"deleteRule": null,
				"fields": [
					{
						"autogeneratePattern": "[a-z0-9]{15}",
						"help": "",
						"hidden": false,
						"id": "text3208210256",
						"max": 15,
						"min": 15,
						"name": "id",
						"pattern": "^[a-z0-9]+$",
						"presentable": false,
						"primaryKey": true,
						"required": true,
						"system": true,
						"type": "text"
					},
					{
						"autogeneratePattern": "",
						"help": "",
						"hidden": false,
						"id": "text455797646",
						"max": 0,
						"min": 0,
						"name": "collectionRef",
						"pattern": "",
						"presentable": false,
						"primaryKey": false,
						"required": true,
						"system": true,
						"type": "text"
					},
					{
						"autogeneratePattern": "",
						"help": "",
						"hidden": false,
						"id": "text127846527",
						"max": 0,
						"min": 0,
						"name": "recordRef",
						"pattern": "",
						"presentable": false,
						"primaryKey": false,
						"required": true,
						"system": true,
						"type": "text"
					},
					{
						"autogeneratePattern": "",
						"help": "",
						"hidden": false,
						"id": "text1582905952",
						"max": 0,
						"min": 0,
						"name": "method",
						"pattern": "",
						"presentable": false,
						"primaryKey": false,
						"required": true,
						"system": true,
						"type": "text"
					},
					{
						"hidden": false,
						"id": "autodate2990389176",
						"name": "created",
						"onCreate": true,
						"onUpdate": false,
						"presentable": false,
						"system": true,
						"type": "autodate"
					},
					{
						"hidden": false,
						"id": "autodate3332085495",
						"name": "updated",
						"onCreate": true,
						"onUpdate": true,
						"presentable": false,
						"system": true,
						"type": "autodate"
					}
				],
				"id": "pbc_2279338944",
				"indexes": [
					"CREATE INDEX ` + "`" + `idx_mfas_collectionRef_recordRef` + "`" + ` ON ` + "`" + `_mfas` + "`" + ` (collectionRef,recordRef)"
				],
				"listRule": "@request.auth.id != '' && recordRef = @request.auth.id && collectionRef = @request.auth.collectionId",
				"name": "_mfas",
				"system": true,
				"type": "base",
				"updateRule": null,
				"viewRule": "@request.auth.id != '' && recordRef = @request.auth.id && collectionRef = @request.auth.collectionId"
			},
			{
				"createRule": null,
				"deleteRule": null,
				"fields": [
					{
						"autogeneratePattern": "[a-z0-9]{15}",
						"help": "",
						"hidden": false,
						"id": "text3208210256",
						"max": 15,
						"min": 15,
						"name": "id",
						"pattern": "^[a-z0-9]+$",
						"presentable": false,
						"primaryKey": true,
						"required": true,
						"system": true,
						"type": "text"
					},
					{
						"autogeneratePattern": "",
						"help": "",
						"hidden": false,
						"id": "text455797646",
						"max": 0,
						"min": 0,
						"name": "collectionRef",
						"pattern": "",
						"presentable": false,
						"primaryKey": false,
						"required": true,
						"system": true,
						"type": "text"
					},
					{
						"autogeneratePattern": "",
						"help": "",
						"hidden": false,
						"id": "text127846527",
						"max": 0,
						"min": 0,
						"name": "recordRef",
						"pattern": "",
						"presentable": false,
						"primaryKey": false,
						"required": true,
						"system": true,
						"type": "text"
					},
					{
						"cost": 8,
						"help": "",
						"hidden": true,
						"id": "password901924565",
						"max": 0,
						"min": 0,
						"name": "password",
						"pattern": "",
						"presentable": false,
						"required": true,
						"system": true,
						"type": "password"
					},
					{
						"autogeneratePattern": "",
						"help": "",
						"hidden": true,
						"id": "text3866985172",
						"max": 0,
						"min": 0,
						"name": "sentTo",
						"pattern": "",
						"presentable": false,
						"primaryKey": false,
						"required": false,
						"system": true,
						"type": "text"
					},
					{
						"hidden": false,
						"id": "autodate2990389176",
						"name": "created",
						"onCreate": true,
						"onUpdate": false,
						"presentable": false,
						"system": true,
						"type": "autodate"
					},
					{
						"hidden": false,
						"id": "autodate3332085495",
						"name": "updated",
						"onCreate": true,
						"onUpdate": true,
						"presentable": false,
						"system": true,
						"type": "autodate"
					}
				],
				"id": "pbc_1638494021",
				"indexes": [
					"CREATE INDEX ` + "`" + `idx_otps_collectionRef_recordRef` + "`" + ` ON ` + "`" + `_otps` + "`" + ` (collectionRef, recordRef)"
				],
				"listRule": "@request.auth.id != '' && recordRef = @request.auth.id && collectionRef = @request.auth.collectionId",
				"name": "_otps",
				"system": true,
				"type": "base",
				"updateRule": null,
				"viewRule": "@request.auth.id != '' && recordRef = @request.auth.id && collectionRef = @request.auth.collectionId"
			},
			{
				"createRule": null,
				"deleteRule": "@request.auth.id != '' && recordRef = @request.auth.id && collectionRef = @request.auth.collectionId",
				"fields": [
					{
						"autogeneratePattern": "[a-z0-9]{15}",
						"help": "",
						"hidden": false,
						"id": "text3208210256",
						"max": 15,
						"min": 15,
						"name": "id",
						"pattern": "^[a-z0-9]+$",
						"presentable": false,
						"primaryKey": true,
						"required": true,
						"system": true,
						"type": "text"
					},
					{
						"autogeneratePattern": "",
						"help": "",
						"hidden": false,
						"id": "text455797646",
						"max": 0,
						"min": 0,
						"name": "collectionRef",
						"pattern": "",
						"presentable": false,
						"primaryKey": false,
						"required": true,
						"system": true,
						"type": "text"
					},
					{
						"autogeneratePattern": "",
						"help": "",
						"hidden": false,
						"id": "text127846527",
						"max": 0,
						"min": 0,
						"name": "recordRef",
						"pattern": "",
						"presentable": false,
						"primaryKey": false,
						"required": true,
						"system": true,
						"type": "text"
					},
					{
						"autogeneratePattern": "",
						"help": "",
						"hidden": false,
						"id": "text2462348188",
						"max": 0,
						"min": 0,
						"name": "provider",
						"pattern": "",
						"presentable": false,
						"primaryKey": false,
						"required": true,
						"system": true,
						"type": "text"
					},
					{
						"autogeneratePattern": "",
						"help": "",
						"hidden": false,
						"id": "text1044722854",
						"max": 0,
						"min": 0,
						"name": "providerId",
						"pattern": "",
						"presentable": false,
						"primaryKey": false,
						"required": true,
						"system": true,
						"type": "text"
					},
					{
						"hidden": false,
						"id": "autodate2990389176",
						"name": "created",
						"onCreate": true,
						"onUpdate": false,
						"presentable": false,
						"system": true,
						"type": "autodate"
					},
					{
						"hidden": false,
						"id": "autodate3332085495",
						"name": "updated",
						"onCreate": true,
						"onUpdate": true,
						"presentable": false,
						"system": true,
						"type": "autodate"
					}
				],
				"id": "pbc_2281828961",
				"indexes": [
					"CREATE UNIQUE INDEX ` + "`" + `idx_externalAuths_record_provider` + "`" + ` ON ` + "`" + `_externalAuths` + "`" + ` (collectionRef, recordRef, provider)",
					"CREATE UNIQUE INDEX ` + "`" + `idx_externalAuths_collection_provider` + "`" + ` ON ` + "`" + `_externalAuths` + "`" + ` (collectionRef, provider, providerId)"
				],
				"listRule": "@request.auth.id != '' && recordRef = @request.auth.id && collectionRef = @request.auth.collectionId",
				"name": "_externalAuths",
				"system": true,
				"type": "base",
				"updateRule": null,
				"viewRule": "@request.auth.id != '' && recordRef = @request.auth.id && collectionRef = @request.auth.collectionId"
			},
			{
				"createRule": null,
				"deleteRule": "@request.auth.id != '' && recordRef = @request.auth.id && collectionRef = @request.auth.collectionId",
				"fields": [
					{
						"autogeneratePattern": "[a-z0-9]{15}",
						"help": "",
						"hidden": false,
						"id": "text3208210256",
						"max": 15,
						"min": 15,
						"name": "id",
						"pattern": "^[a-z0-9]+$",
						"presentable": false,
						"primaryKey": true,
						"required": true,
						"system": true,
						"type": "text"
					},
					{
						"autogeneratePattern": "",
						"help": "",
						"hidden": false,
						"id": "text455797646",
						"max": 0,
						"min": 0,
						"name": "collectionRef",
						"pattern": "",
						"presentable": false,
						"primaryKey": false,
						"required": true,
						"system": true,
						"type": "text"
					},
					{
						"autogeneratePattern": "",
						"help": "",
						"hidden": false,
						"id": "text127846527",
						"max": 0,
						"min": 0,
						"name": "recordRef",
						"pattern": "",
						"presentable": false,
						"primaryKey": false,
						"required": true,
						"system": true,
						"type": "text"
					},
					{
						"autogeneratePattern": "",
						"help": "",
						"hidden": false,
						"id": "text4228609354",
						"max": 0,
						"min": 0,
						"name": "fingerprint",
						"pattern": "",
						"presentable": false,
						"primaryKey": false,
						"required": true,
						"system": true,
						"type": "text"
					},
					{
						"hidden": false,
						"id": "autodate2990389176",
						"name": "created",
						"onCreate": true,
						"onUpdate": false,
						"presentable": false,
						"system": true,
						"type": "autodate"
					},
					{
						"hidden": false,
						"id": "autodate3332085495",
						"name": "updated",
						"onCreate": true,
						"onUpdate": true,
						"presentable": false,
						"system": true,
						"type": "autodate"
					}
				],
				"id": "pbc_4275539003",
				"indexes": [
					"CREATE UNIQUE INDEX ` + "`" + `idx_authOrigins_unique_pairs` + "`" + ` ON ` + "`" + `_authOrigins` + "`" + ` (collectionRef, recordRef, fingerprint)"
				],
				"listRule": "@request.auth.id != '' && recordRef = @request.auth.id && collectionRef = @request.auth.collectionId",
				"name": "_authOrigins",
				"system": true,
				"type": "base",
				"updateRule": null,
				"viewRule": "@request.auth.id != '' && recordRef = @request.auth.id && collectionRef = @request.auth.collectionId"
			},
			{
				"authAlert": {
					"emailTemplate": {
						"body": "<p>Hello,</p>\n<p>We noticed a login to your {APP_NAME} account from a new location:</p>\n<p><em>{ALERT_INFO}</em></p>\n<p><strong>If this wasn't you, you should immediately change your {APP_NAME} account password to revoke access from all other locations.</strong></p>\n<p>If this was you, you may disregard this email.</p>\n<p>\n  Thanks,<br/>\n  {APP_NAME} team\n</p>",
						"subject": "Login from a new location"
					},
					"enabled": true
				},
				"authRule": "",
				"authToken": {
					"duration": 86400
				},
				"confirmEmailChangeTemplate": {
					"body": "<p>Hello,</p>\n<p>Click on the button below to confirm your new email address.</p>\n<p>\n  <a class=\"btn\" href=\"{APP_URL}/_/#/auth/confirm-email-change/{TOKEN}\" target=\"_blank\" rel=\"noopener\">Confirm new email</a>\n</p>\n<p><i>If you didn't ask to change your email address, please ignore this email.</i></p>\n<p>\n  Thanks,<br/>\n  {APP_NAME} team\n</p>",
					"subject": "Confirm your {APP_NAME} new email address"
				},
				"createRule": null,
				"deleteRule": null,
				"emailChangeToken": {
					"duration": 1800
				},
				"fields": [
					{
						"autogeneratePattern": "[a-z0-9]{15}",
						"help": "",
						"hidden": false,
						"id": "text3208210256",
						"max": 15,
						"min": 15,
						"name": "id",
						"pattern": "^[a-z0-9]+$",
						"presentable": false,
						"primaryKey": true,
						"required": true,
						"system": true,
						"type": "text"
					},
					{
						"cost": 0,
						"help": "",
						"hidden": true,
						"id": "password901924565",
						"max": 0,
						"min": 8,
						"name": "password",
						"pattern": "",
						"presentable": false,
						"required": true,
						"system": true,
						"type": "password"
					},
					{
						"autogeneratePattern": "[a-zA-Z0-9]{50}",
						"help": "",
						"hidden": true,
						"id": "text2504183744",
						"max": 60,
						"min": 30,
						"name": "tokenKey",
						"pattern": "",
						"presentable": false,
						"primaryKey": false,
						"required": true,
						"system": true,
						"type": "text"
					},
					{
						"exceptDomains": null,
						"help": "",
						"hidden": false,
						"id": "email3885137012",
						"name": "email",
						"onlyDomains": null,
						"presentable": false,
						"required": true,
						"system": true,
						"type": "email"
					},
					{
						"help": "",
						"hidden": false,
						"id": "bool1547992806",
						"name": "emailVisibility",
						"presentable": false,
						"required": false,
						"system": true,
						"type": "bool"
					},
					{
						"help": "",
						"hidden": false,
						"id": "bool256245529",
						"name": "verified",
						"presentable": false,
						"required": false,
						"system": true,
						"type": "bool"
					},
					{
						"hidden": false,
						"id": "autodate2990389176",
						"name": "created",
						"onCreate": true,
						"onUpdate": false,
						"presentable": false,
						"system": true,
						"type": "autodate"
					},
					{
						"hidden": false,
						"id": "autodate3332085495",
						"name": "updated",
						"onCreate": true,
						"onUpdate": true,
						"presentable": false,
						"system": true,
						"type": "autodate"
					}
				],
				"fileToken": {
					"duration": 180
				},
				"id": "pbc_3142635823",
				"indexes": [
					"CREATE UNIQUE INDEX ` + "`" + `idx_tokenKey_pbc_3142635823` + "`" + ` ON ` + "`" + `_superusers` + "`" + ` (` + "`" + `tokenKey` + "`" + `)",
					"CREATE UNIQUE INDEX ` + "`" + `idx_email_pbc_3142635823` + "`" + ` ON ` + "`" + `_superusers` + "`" + ` (` + "`" + `email` + "`" + `) WHERE ` + "`" + `email` + "`" + ` != ''"
				],
				"listRule": null,
				"manageRule": null,
				"mfa": {
					"duration": 600,
					"enabled": false,
					"rule": ""
				},
				"name": "_superusers",
				"oauth2": {
					"enabled": false,
					"mappedFields": {
						"avatarURL": "",
						"id": "",
						"name": "",
						"username": ""
					}
				},
				"otp": {
					"duration": 180,
					"emailTemplate": {
						"body": "<p>Hello,</p>\n<p>Your one-time password is: <strong>{OTP}</strong></p>\n<p><i>If you didn't ask for the one-time password, you can ignore this email.</i></p>\n<p>\n  Thanks,<br/>\n  {APP_NAME} team\n</p>",
						"subject": "OTP for {APP_NAME}"
					},
					"enabled": false,
					"length": 8
				},
				"passwordAuth": {
					"enabled": true,
					"identityFields": [
						"email"
					]
				},
				"passwordResetToken": {
					"duration": 1800
				},
				"resetPasswordTemplate": {
					"body": "<p>Hello,</p>\n<p>Click on the button below to reset your password.</p>\n<p>\n  <a class=\"btn\" href=\"{APP_URL}/_/#/auth/confirm-password-reset/{TOKEN}\" target=\"_blank\" rel=\"noopener\">Reset password</a>\n</p>\n<p><i>If you didn't ask to reset your password, please ignore this email.</i></p>\n<p>\n  Thanks,<br/>\n  {APP_NAME} team\n</p>",
					"subject": "Reset your {APP_NAME} password"
				},
				"system": true,
				"type": "auth",
				"updateRule": null,
				"verificationTemplate": {
					"body": "<p>Hello,</p>\n<p>Thank you for joining us at {APP_NAME}.</p>\n<p>Click on the button below to verify your email address.</p>\n<p>\n  <a class=\"btn\" href=\"{APP_URL}/_/#/auth/confirm-verification/{TOKEN}\" target=\"_blank\" rel=\"noopener\">Verify</a>\n</p>\n<p><i>If you didn't recently register, please ignore this email.</i></p>\n<p>\n  Thanks,<br/>\n  {APP_NAME} team\n</p>",
					"subject": "Verify your {APP_NAME} email"
				},
				"verificationToken": {
					"duration": 86400
				},
				"viewRule": null
			},
			{
				"authAlert": {
					"emailTemplate": {
						"body": "<p>Hello,</p>\n<p>We noticed a login to your {APP_NAME} account from a new location:</p>\n<p><em>{ALERT_INFO}</em></p>\n<p><strong>If this wasn't you, you should immediately change your {APP_NAME} account password to revoke access from all other locations.</strong></p>\n<p>If this was you, you may disregard this email.</p>\n<p>\n  Thanks,<br/>\n  {APP_NAME} team\n</p>",
						"subject": "Login from a new location"
					},
					"enabled": true
				},
				"authRule": "",
				"authToken": {
					"duration": 432000
				},
				"confirmEmailChangeTemplate": {
					"body": "<p>Hello,</p>\n<p>Click on the button below to confirm your new email address.</p>\n<p>\n  <a class=\"btn\" href=\"{APP_URL}/_/#/auth/confirm-email-change/{TOKEN}\" target=\"_blank\" rel=\"noopener\">Confirm new email</a>\n</p>\n<p><i>If you didn't ask to change your email address, please ignore this email.</i></p>\n<p>\n  Thanks,<br/>\n  {APP_NAME} team\n</p>",
					"subject": "Confirm your {APP_NAME} new email address"
				},
				"createRule": "",
				"deleteRule": "id = @request.auth.id",
				"emailChangeToken": {
					"duration": 1800
				},
				"fields": [
					{
						"autogeneratePattern": "[a-z0-9]{15}",
						"help": "",
						"hidden": false,
						"id": "text3208210256",
						"max": 15,
						"min": 15,
						"name": "id",
						"pattern": "^[a-z0-9]+$",
						"presentable": false,
						"primaryKey": true,
						"required": true,
						"system": true,
						"type": "text"
					},
					{
						"cost": 0,
						"help": "",
						"hidden": true,
						"id": "password901924565",
						"max": 0,
						"min": 8,
						"name": "password",
						"pattern": "",
						"presentable": false,
						"required": true,
						"system": true,
						"type": "password"
					},
					{
						"autogeneratePattern": "[a-zA-Z0-9]{50}",
						"help": "",
						"hidden": true,
						"id": "text2504183744",
						"max": 60,
						"min": 30,
						"name": "tokenKey",
						"pattern": "",
						"presentable": false,
						"primaryKey": false,
						"required": true,
						"system": true,
						"type": "text"
					},
					{
						"exceptDomains": null,
						"help": "",
						"hidden": false,
						"id": "email3885137012",
						"name": "email",
						"onlyDomains": null,
						"presentable": false,
						"required": true,
						"system": true,
						"type": "email"
					},
					{
						"help": "",
						"hidden": false,
						"id": "bool1547992806",
						"name": "emailVisibility",
						"presentable": false,
						"required": false,
						"system": true,
						"type": "bool"
					},
					{
						"help": "",
						"hidden": false,
						"id": "bool256245529",
						"name": "verified",
						"presentable": false,
						"required": false,
						"system": true,
						"type": "bool"
					},
					{
						"autogeneratePattern": "",
						"help": "",
						"hidden": false,
						"id": "text1579384326",
						"max": 255,
						"min": 0,
						"name": "name",
						"pattern": "",
						"presentable": false,
						"primaryKey": false,
						"required": false,
						"system": false,
						"type": "text"
					},
					{
						"help": "",
						"hidden": false,
						"id": "file376926767",
						"maxSelect": 1,
						"maxSize": 0,
						"mimeTypes": [
							"image/jpeg",
							"image/png",
							"image/svg+xml",
							"image/gif",
							"image/webp"
						],
						"name": "avatar",
						"presentable": false,
						"protected": false,
						"required": false,
						"system": false,
						"thumbs": null,
						"type": "file"
					},
					{
						"hidden": false,
						"id": "autodate2990389176",
						"name": "created",
						"onCreate": true,
						"onUpdate": false,
						"presentable": false,
						"system": false,
						"type": "autodate"
					},
					{
						"hidden": false,
						"id": "autodate3332085495",
						"name": "updated",
						"onCreate": true,
						"onUpdate": true,
						"presentable": false,
						"system": false,
						"type": "autodate"
					}
				],
				"fileToken": {
					"duration": 180
				},
				"id": "_pb_users_auth_",
				"indexes": [
					"CREATE UNIQUE INDEX ` + "`" + `idx_tokenKey__pb_users_auth_` + "`" + ` ON ` + "`" + `users` + "`" + ` (` + "`" + `tokenKey` + "`" + `)",
					"CREATE UNIQUE INDEX ` + "`" + `idx_email__pb_users_auth_` + "`" + ` ON ` + "`" + `users` + "`" + ` (` + "`" + `email` + "`" + `) WHERE ` + "`" + `email` + "`" + ` != ''"
				],
				"listRule": "id = @request.auth.id",
				"manageRule": null,
				"mfa": {
					"duration": 600,
					"enabled": false,
					"rule": ""
				},
				"name": "users",
				"oauth2": {
					"enabled": false,
					"mappedFields": {
						"avatarURL": "avatar",
						"id": "",
						"name": "name",
						"username": ""
					}
				},
				"otp": {
					"duration": 180,
					"emailTemplate": {
						"body": "<p>Hello,</p>\n<p>Your one-time password is: <strong>{OTP}</strong></p>\n<p><i>If you didn't ask for the one-time password, you can ignore this email.</i></p>\n<p>\n  Thanks,<br/>\n  {APP_NAME} team\n</p>",
						"subject": "OTP for {APP_NAME}"
					},
					"enabled": false,
					"length": 8
				},
				"passwordAuth": {
					"enabled": true,
					"identityFields": [
						"email"
					]
				},
				"passwordResetToken": {
					"duration": 1800
				},
				"resetPasswordTemplate": {
					"body": "<p>Hello,</p>\n<p>Click on the button below to reset your password.</p>\n<p>\n  <a class=\"btn\" href=\"{APP_URL}/_/#/auth/confirm-password-reset/{TOKEN}\" target=\"_blank\" rel=\"noopener\">Reset password</a>\n</p>\n<p><i>If you didn't ask to reset your password, please ignore this email.</i></p>\n<p>\n  Thanks,<br/>\n  {APP_NAME} team\n</p>",
					"subject": "Reset your {APP_NAME} password"
				},
				"system": false,
				"type": "auth",
				"updateRule": "id = @request.auth.id",
				"verificationTemplate": {
					"body": "<p>Hello,</p>\n<p>Thank you for joining us at {APP_NAME}.</p>\n<p>Click on the button below to verify your email address.</p>\n<p>\n  <a class=\"btn\" href=\"{APP_URL}/_/#/auth/confirm-verification/{TOKEN}\" target=\"_blank\" rel=\"noopener\">Verify</a>\n</p>\n<p><i>If you didn't recently register, please ignore this email.</i></p>\n<p>\n  Thanks,<br/>\n  {APP_NAME} team\n</p>",
					"subject": "Verify your {APP_NAME} email"
				},
				"verificationToken": {
					"duration": 86400
				},
				"viewRule": "id = @request.auth.id"
			},
			{
				"createRule": null,
				"deleteRule": null,
				"fields": [
					{
						"autogeneratePattern": "[a-z0-9]{15}",
						"help": "",
						"hidden": false,
						"id": "text3208210256",
						"max": 15,
						"min": 15,
						"name": "id",
						"pattern": "^[a-z0-9]+$",
						"presentable": false,
						"primaryKey": true,
						"required": true,
						"system": true,
						"type": "text"
					},
					{
						"autogeneratePattern": "",
						"help": "",
						"hidden": false,
						"id": "text1032544282",
						"max": 0,
						"min": 0,
						"name": "card_hash",
						"pattern": "",
						"presentable": true,
						"primaryKey": false,
						"required": true,
						"system": false,
						"type": "text"
					},
					{
						"autogeneratePattern": "",
						"help": "",
						"hidden": false,
						"id": "text1007109329",
						"max": 0,
						"min": 0,
						"name": "added_at",
						"pattern": "",
						"presentable": false,
						"primaryKey": false,
						"required": true,
						"system": false,
						"type": "text"
					},
					{
						"autogeneratePattern": "",
						"help": "",
						"hidden": false,
						"id": "text2424154933",
						"max": 0,
						"min": 0,
						"name": "last_reviewed_at",
						"pattern": "",
						"presentable": false,
						"primaryKey": false,
						"required": false,
						"system": false,
						"type": "text"
					},
					{
						"help": "",
						"hidden": false,
						"id": "number2946385302",
						"max": null,
						"min": null,
						"name": "stability",
						"onlyInt": false,
						"presentable": false,
						"required": false,
						"system": false,
						"type": "number"
					},
					{
						"help": "",
						"hidden": false,
						"id": "number3144380399",
						"max": null,
						"min": null,
						"name": "difficulty",
						"onlyInt": false,
						"presentable": false,
						"required": false,
						"system": false,
						"type": "number"
					},
					{
						"help": "",
						"hidden": false,
						"id": "number2917262397",
						"max": null,
						"min": null,
						"name": "interval_raw",
						"onlyInt": false,
						"presentable": false,
						"required": false,
						"system": false,
						"type": "number"
					},
					{
						"help": "",
						"hidden": false,
						"id": "number2822391603",
						"max": null,
						"min": null,
						"name": "interval_days",
						"onlyInt": true,
						"presentable": false,
						"required": false,
						"system": false,
						"type": "number"
					},
					{
						"autogeneratePattern": "",
						"help": "",
						"hidden": false,
						"id": "text3866337329",
						"max": 0,
						"min": 0,
						"name": "due_date",
						"pattern": "",
						"presentable": false,
						"primaryKey": false,
						"required": false,
						"system": false,
						"type": "text"
					},
					{
						"help": "",
						"hidden": false,
						"id": "number2522310777",
						"max": null,
						"min": null,
						"name": "review_count",
						"onlyInt": true,
						"presentable": false,
						"required": false,
						"system": false,
						"type": "number"
					}
				],
				"id": "pbc_3481593366",
				"indexes": [
					"CREATE UNIQUE INDEX ` + "`" + `idx_cards_card_hash_unique` + "`" + ` ON ` + "`" + `cards` + "`" + ` (card_hash)"
				],
				"listRule": null,
				"name": "cards",
				"system": false,
				"type": "base",
				"updateRule": null,
				"viewRule": null
			},
			{
				"createRule": null,
				"deleteRule": null,
				"fields": [
					{
						"autogeneratePattern": "[a-z0-9]{15}",
						"help": "",
						"hidden": false,
						"id": "text3208210256",
						"max": 15,
						"min": 15,
						"name": "id",
						"pattern": "^[a-z0-9]+$",
						"presentable": false,
						"primaryKey": true,
						"required": true,
						"system": true,
						"type": "text"
					},
					{
						"autogeneratePattern": "",
						"help": "",
						"hidden": false,
						"id": "text1631579359",
						"max": 0,
						"min": 0,
						"name": "session_id",
						"pattern": "",
						"presentable": false,
						"primaryKey": false,
						"required": true,
						"system": false,
						"type": "text"
					},
					{
						"autogeneratePattern": "",
						"help": "",
						"hidden": false,
						"id": "text1032544282",
						"max": 0,
						"min": 0,
						"name": "card_hash",
						"pattern": "",
						"presentable": false,
						"primaryKey": false,
						"required": true,
						"system": false,
						"type": "text"
					},
					{
						"autogeneratePattern": "",
						"help": "",
						"hidden": false,
						"id": "text3494630457",
						"max": 0,
						"min": 0,
						"name": "reviewed_at",
						"pattern": "",
						"presentable": false,
						"primaryKey": false,
						"required": true,
						"system": false,
						"type": "text"
					},
					{
						"autogeneratePattern": "",
						"help": "",
						"hidden": false,
						"id": "text1499115060",
						"max": 0,
						"min": 0,
						"name": "grade",
						"pattern": "",
						"presentable": false,
						"primaryKey": false,
						"required": true,
						"system": false,
						"type": "text"
					},
					{
						"help": "",
						"hidden": false,
						"id": "number2946385302",
						"max": null,
						"min": null,
						"name": "stability",
						"onlyInt": false,
						"presentable": false,
						"required": true,
						"system": false,
						"type": "number"
					},
					{
						"help": "",
						"hidden": false,
						"id": "number3144380399",
						"max": null,
						"min": null,
						"name": "difficulty",
						"onlyInt": false,
						"presentable": false,
						"required": true,
						"system": false,
						"type": "number"
					},
					{
						"help": "",
						"hidden": false,
						"id": "number2917262397",
						"max": null,
						"min": null,
						"name": "interval_raw",
						"onlyInt": false,
						"presentable": false,
						"required": true,
						"system": false,
						"type": "number"
					},
					{
						"help": "",
						"hidden": false,
						"id": "number2822391603",
						"max": null,
						"min": null,
						"name": "interval_days",
						"onlyInt": true,
						"presentable": false,
						"required": true,
						"system": false,
						"type": "number"
					},
					{
						"autogeneratePattern": "",
						"help": "",
						"hidden": false,
						"id": "text3866337329",
						"max": 0,
						"min": 0,
						"name": "due_date",
						"pattern": "",
						"presentable": false,
						"primaryKey": false,
						"required": true,
						"system": false,
						"type": "text"
					}
				],
				"id": "pbc_4163081445",
				"indexes": [
					"CREATE INDEX ` + "`" + `idx_reviews_session_id` + "`" + ` ON ` + "`" + `reviews` + "`" + ` (session_id)",
					"CREATE INDEX ` + "`" + `idx_reviews_card_hash` + "`" + ` ON ` + "`" + `reviews` + "`" + ` (card_hash)"
				],
				"listRule": null,
				"name": "reviews",
				"system": false,
				"type": "base",
				"updateRule": null,
				"viewRule": null
			},
			{
				"createRule": null,
				"deleteRule": null,
				"fields": [
					{
						"autogeneratePattern": "[a-z0-9]{15}",
						"help": "",
						"hidden": false,
						"id": "text3208210256",
						"max": 15,
						"min": 15,
						"name": "id",
						"pattern": "^[a-z0-9]+$",
						"presentable": false,
						"primaryKey": true,
						"required": true,
						"system": true,
						"type": "text"
					},
					{
						"autogeneratePattern": "",
						"help": "",
						"hidden": false,
						"id": "text222754019",
						"max": 0,
						"min": 0,
						"name": "started_at",
						"pattern": "",
						"presentable": false,
						"primaryKey": false,
						"required": true,
						"system": false,
						"type": "text"
					},
					{
						"autogeneratePattern": "",
						"help": "",
						"hidden": false,
						"id": "text473765221",
						"max": 0,
						"min": 0,
						"name": "ended_at",
						"pattern": "",
						"presentable": false,
						"primaryKey": false,
						"required": true,
						"system": false,
						"type": "text"
					}
				],
				"id": "pbc_3660498186",
				"indexes": [
					"CREATE INDEX ` + "`" + `idx_sessions_started_at` + "`" + ` ON ` + "`" + `sessions` + "`" + ` (started_at)"
				],
				"listRule": null,
				"name": "sessions",
				"system": false,
				"type": "base",
				"updateRule": null,
				"viewRule": null
			},
			{
				"createRule": null,
				"deleteRule": null,
				"fields": [
					{
						"autogeneratePattern": "[a-z0-9]{15}",
						"help": "",
						"hidden": false,
						"id": "text3208210256",
						"max": 15,
						"min": 15,
						"name": "id",
						"pattern": "^[a-z0-9]+$",
						"presentable": false,
						"primaryKey": true,
						"required": true,
						"system": true,
						"type": "text"
					},
					{
						"cascadeDelete": false,
						"collectionId": "pbc_3481593366",
						"help": "",
						"hidden": false,
						"id": "relation1653163849",
						"maxSelect": 0,
						"minSelect": 0,
						"name": "relation",
						"presentable": false,
						"required": false,
						"system": false,
						"type": "relation"
					},
					{
						"autogeneratePattern": "",
						"help": "",
						"hidden": false,
						"id": "text1657027128",
						"max": 0,
						"min": 0,
						"name": "deck_name",
						"pattern": "",
						"presentable": false,
						"primaryKey": false,
						"required": false,
						"system": false,
						"type": "text"
					},
					{
						"help": "",
						"hidden": false,
						"id": "json3036287351",
						"maxSize": 0,
						"name": "card_body",
						"presentable": false,
						"required": false,
						"system": false,
						"type": "json"
					}
				],
				"id": "pbc_3446931122",
				"indexes": [],
				"listRule": null,
				"name": "files",
				"system": false,
				"type": "base",
				"updateRule": null,
				"viewRule": null
			},
			{
				"createRule": null,
				"deleteRule": null,
				"fields": [
					{
						"autogeneratePattern": "",
						"help": "",
						"hidden": false,
						"id": "text3208210256",
						"max": 0,
						"min": 0,
						"name": "id",
						"pattern": "^[a-z0-9]+$",
						"presentable": false,
						"primaryKey": true,
						"required": true,
						"system": true,
						"type": "text"
					},
					{
						"autogeneratePattern": "",
						"help": "",
						"hidden": false,
						"id": "_clone_0GaS",
						"max": 0,
						"min": 0,
						"name": "deck_name",
						"pattern": "",
						"presentable": false,
						"primaryKey": false,
						"required": false,
						"system": false,
						"type": "text"
					},
					{
						"help": "",
						"hidden": false,
						"id": "number4275314957",
						"max": null,
						"min": null,
						"name": "day_offset",
						"onlyInt": true,
						"presentable": false,
						"required": false,
						"system": false,
						"type": "number"
					},
					{
						"help": "",
						"hidden": false,
						"id": "number2245608546",
						"max": null,
						"min": null,
						"name": "count",
						"onlyInt": true,
						"presentable": false,
						"required": false,
						"system": false,
						"type": "number"
					}
				],
				"id": "pbc_2005503171",
				"indexes": [],
				"listRule": null,
				"name": "future_due",
				"system": false,
				"type": "view",
				"updateRule": null,
				"viewQuery": "SELECT\n  (ROW_NUMBER() OVER()) AS id,\n  f.deck_name,\n  CAST(julianday(date(c.due_date)) - julianday(date('now')) AS INTEGER) AS day_offset,\n  COUNT(*) AS count\nFROM cards c\nJOIN files f ON f.relation = c.id\nWHERE c.due_date IS NOT NULL AND c.due_date != ''\n  AND date(c.due_date) >= date('now')\n  AND date(c.due_date) < date('now', '+365 days')\nGROUP BY f.deck_name, day_offset\nORDER BY f.deck_name, day_offset;",
				"viewRule": null
			},
			{
				"createRule": null,
				"deleteRule": null,
				"fields": [
					{
						"autogeneratePattern": "",
						"help": "",
						"hidden": false,
						"id": "text3208210256",
						"max": 0,
						"min": 0,
						"name": "id",
						"pattern": "^[a-z0-9]+$",
						"presentable": false,
						"primaryKey": true,
						"required": true,
						"system": true,
						"type": "text"
					},
					{
						"autogeneratePattern": "",
						"help": "",
						"hidden": false,
						"id": "_clone_zczB",
						"max": 0,
						"min": 0,
						"name": "deck_name",
						"pattern": "",
						"presentable": false,
						"primaryKey": false,
						"required": false,
						"system": false,
						"type": "text"
					},
					{
						"help": "",
						"hidden": false,
						"id": "number2102331327",
						"max": null,
						"min": null,
						"name": "new_count",
						"onlyInt": true,
						"presentable": false,
						"required": false,
						"system": false,
						"type": "number"
					},
					{
						"help": "",
						"hidden": false,
						"id": "number1687892461",
						"max": null,
						"min": null,
						"name": "due_count",
						"onlyInt": true,
						"presentable": false,
						"required": false,
						"system": false,
						"type": "number"
					},
					{
						"help": "",
						"hidden": false,
						"id": "number1745672659",
						"max": null,
						"min": null,
						"name": "reviewed_today_count",
						"onlyInt": true,
						"presentable": false,
						"required": false,
						"system": false,
						"type": "number"
					}
				],
				"id": "pbc_273357442",
				"indexes": [],
				"listRule": null,
				"name": "today_status",
				"system": false,
				"type": "view",
				"updateRule": null,
				"viewQuery": "SELECT\n    (ROW_NUMBER() OVER()) AS id,   \n  f.deck_name AS deck_name,   \n  CAST(SUM(CASE WHEN c.due_date = '' THEN 1 ELSE 0 END) AS INT) AS \nnew_count,\n  CAST(SUM(CASE WHEN c.due_date != '' AND date(c.due_date) <= \ndate('now') THEN 1 ELSE 0 END) AS INT) AS due_count,\n  CAST(SUM(CASE WHEN c.last_reviewed_at != '' AND \ndate(c.last_reviewed_at) = date('now') THEN 1 ELSE 0 END) AS INT) AS \nreviewed_today_count\nFROM cards c\nJOIN files f ON f.relation = c.id\nGROUP BY f.deck_name\nORDER BY f.deck_name;\n",
				"viewRule": null
			},
			{
				"createRule": null,
				"deleteRule": null,
				"fields": [
					{
						"autogeneratePattern": "",
						"help": "",
						"hidden": false,
						"id": "text3208210256",
						"max": 0,
						"min": 0,
						"name": "id",
						"pattern": "^[a-z0-9]+$",
						"presentable": false,
						"primaryKey": true,
						"required": true,
						"system": true,
						"type": "text"
					},
					{
						"autogeneratePattern": "",
						"help": "",
						"hidden": false,
						"id": "text1657027128",
						"max": 0,
						"min": 0,
						"name": "deck_name",
						"pattern": "",
						"presentable": false,
						"primaryKey": false,
						"required": false,
						"system": false,
						"type": "text"
					},
					{
						"help": "",
						"hidden": false,
						"id": "number2102331327",
						"max": null,
						"min": null,
						"name": "new_count",
						"onlyInt": true,
						"presentable": false,
						"required": false,
						"system": false,
						"type": "number"
					},
					{
						"help": "",
						"hidden": false,
						"id": "number1687892461",
						"max": null,
						"min": null,
						"name": "due_count",
						"onlyInt": true,
						"presentable": false,
						"required": false,
						"system": false,
						"type": "number"
					},
					{
						"help": "",
						"hidden": false,
						"id": "number1745672659",
						"max": null,
						"min": null,
						"name": "reviewed_today_count",
						"onlyInt": true,
						"presentable": false,
						"required": false,
						"system": false,
						"type": "number"
					}
				],
				"id": "pbc_3108102281",
				"indexes": [],
				"listRule": null,
				"name": "today_status_all",
				"system": false,
				"type": "view",
				"updateRule": null,
				"viewQuery": "SELECT\n  1 AS id,\n  CAST('All' AS TEXT) AS deck_name,\n  CAST(SUM(CASE WHEN c.due_date = '' THEN 1 ELSE 0 END) AS INTEGER) AS new_count,\n  CAST(SUM(CASE WHEN c.due_date != '' AND date(c.due_date) <= date('now') THEN 1 ELSE 0 END) AS INTEGER) AS due_count,\n  CAST(SUM(CASE WHEN c.last_reviewed_at != '' AND date(c.last_reviewed_at) = date('now') THEN 1 ELSE 0 END) AS INTEGER) AS reviewed_today_count\nFROM cards c\nJOIN files f ON f.relation = c.id;",
				"viewRule": null
			},
			{
				"createRule": null,
				"deleteRule": null,
				"fields": [
					{
						"autogeneratePattern": "",
						"help": "",
						"hidden": false,
						"id": "text3208210256",
						"max": 0,
						"min": 0,
						"name": "id",
						"pattern": "^[a-z0-9]+$",
						"presentable": false,
						"primaryKey": true,
						"required": true,
						"system": true,
						"type": "text"
					},
					{
						"help": "",
						"hidden": false,
						"id": "json1657027128",
						"maxSize": 1,
						"name": "deck_name",
						"presentable": false,
						"required": false,
						"system": false,
						"type": "json"
					},
					{
						"help": "",
						"hidden": false,
						"id": "json3030377993",
						"maxSize": 1,
						"name": "retri_pct",
						"presentable": false,
						"required": false,
						"system": false,
						"type": "json"
					}
				],
				"id": "pbc_3514563783",
				"indexes": [],
				"listRule": null,
				"name": "retrievability",
				"system": false,
				"type": "view",
				"updateRule": null,
				"viewQuery": "SELECT\n  (ROW_NUMBER() OVER()) AS id,\n  deck_name,\n  AVG(retrievability) AS retri_pct\nFROM (\n  SELECT\n    f.deck_name AS deck_name,\n    -- New cards (never reviewed) contribute 0, matching computeAvgRetrieval in serve.go.\n    -- Multiplying by 100 here (per-card) instead of after AVG() keeps the\n    -- outer SELECT free of arithmetic on the aggregate result, which is\n    -- required for PocketBase's view column type inference to succeed.\n    CASE\n      WHEN c.last_reviewed_at = '' OR c.stability = 0 THEN 0\n      ELSE (1.0 / sqrt(\n        1.0 + (19.0 / 81.0)\n          * (julianday(date('now')) - julianday(date(c.last_reviewed_at)))\n          / c.stability\n      )) * 100\n    END AS retrievability\n  FROM cards c\n  JOIN files f ON f.relation = c.id\n) card_retrievability\nGROUP BY deck_name\nORDER BY deck_name;",
				"viewRule": null
			},
			{
				"createRule": null,
				"deleteRule": null,
				"fields": [
					{
						"autogeneratePattern": "",
						"help": "",
						"hidden": false,
						"id": "text3208210256",
						"max": 0,
						"min": 0,
						"name": "id",
						"pattern": "^[a-z0-9]+$",
						"presentable": false,
						"primaryKey": true,
						"required": true,
						"system": true,
						"type": "text"
					},
					{
						"autogeneratePattern": "",
						"help": "",
						"hidden": false,
						"id": "text1657027128",
						"max": 0,
						"min": 0,
						"name": "deck_name",
						"pattern": "",
						"presentable": false,
						"primaryKey": false,
						"required": false,
						"system": false,
						"type": "text"
					},
					{
						"help": "",
						"hidden": false,
						"id": "number3030377993",
						"max": null,
						"min": null,
						"name": "retri_pct",
						"onlyInt": true,
						"presentable": false,
						"required": false,
						"system": false,
						"type": "number"
					}
				],
				"id": "pbc_3565123995",
				"indexes": [],
				"listRule": null,
				"name": "retrievability_all",
				"system": false,
				"type": "view",
				"updateRule": null,
				"viewQuery": "SELECT\n  (ROW_NUMBER() OVER()) AS id,\n  CAST('All' AS TEXT) AS deck_name,\n  CAST(AVG(retrievability) AS INTEGER) AS retri_pct\nFROM (\n  SELECT\n    -- New cards (never reviewed) contribute 0, matching computeAvgRetrieval in serve.go.\n    -- Multiplying by 100 here (per-card) instead of after AVG() keeps the\n    -- outer SELECT free of arithmetic on the aggregate result, which is\n    -- required for PocketBase's view column type inference to succeed.\n    CASE\n      WHEN c.last_reviewed_at = '' OR c.stability = 0 THEN 0\n      ELSE (1.0 / sqrt(\n        1.0 + (19.0 / 81.0)\n          * (julianday(date('now')) - julianday(date(c.last_reviewed_at)))\n          / c.stability\n      )) * 100\n    END AS retrievability\n  FROM cards c\n  JOIN files f ON f.relation = c.id\n) card_retrievability;",
				"viewRule": null
			},
			{
				"createRule": null,
				"deleteRule": null,
				"fields": [
					{
						"autogeneratePattern": "",
						"help": "",
						"hidden": false,
						"id": "text3208210256",
						"max": 0,
						"min": 0,
						"name": "id",
						"pattern": "^[a-z0-9]+$",
						"presentable": false,
						"primaryKey": true,
						"required": true,
						"system": true,
						"type": "text"
					},
					{
						"autogeneratePattern": "",
						"help": "",
						"hidden": false,
						"id": "text1657027128",
						"max": 0,
						"min": 0,
						"name": "deck_name",
						"pattern": "",
						"presentable": false,
						"primaryKey": false,
						"required": false,
						"system": false,
						"type": "text"
					},
					{
						"help": "",
						"hidden": false,
						"id": "number4275314957",
						"max": null,
						"min": null,
						"name": "day_offset",
						"onlyInt": true,
						"presentable": false,
						"required": false,
						"system": false,
						"type": "number"
					},
					{
						"help": "",
						"hidden": false,
						"id": "number2245608546",
						"max": null,
						"min": null,
						"name": "count",
						"onlyInt": true,
						"presentable": false,
						"required": false,
						"system": false,
						"type": "number"
					}
				],
				"id": "pbc_1566127584",
				"indexes": [],
				"listRule": null,
				"name": "future_due_all",
				"system": false,
				"type": "view",
				"updateRule": null,
				"viewQuery": "SELECT\n  (ROW_NUMBER() OVER()) AS id,\n  CAST('All' AS TEXT) AS deck_name,\n  CAST(julianday(date(c.due_date)) - julianday(date('now')) AS INTEGER) AS day_offset,\n  COUNT(*) AS count\nFROM cards c\nJOIN files f ON f.relation = c.id\nWHERE c.due_date IS NOT NULL AND c.due_date != ''\n  AND date(c.due_date) >= date('now')\n  AND date(c.due_date) < date('now', '+365 days')\nGROUP BY day_offset\nORDER BY day_offset;",
				"viewRule": null
			},
			{
				"createRule": null,
				"deleteRule": null,
				"fields": [
					{
						"autogeneratePattern": "[a-z0-9]{15}",
						"help": "",
						"hidden": false,
						"id": "text3208210256",
						"max": 15,
						"min": 15,
						"name": "id",
						"pattern": "^[a-z0-9]+$",
						"presentable": false,
						"primaryKey": true,
						"required": true,
						"system": true,
						"type": "text"
					},
					{
						"autogeneratePattern": "",
						"help": "",
						"hidden": false,
						"id": "text1579384326",
						"max": 0,
						"min": 0,
						"name": "name",
						"pattern": "",
						"presentable": false,
						"primaryKey": false,
						"required": true,
						"system": false,
						"type": "text"
					},
					{
						"autogeneratePattern": "",
						"help": "",
						"hidden": false,
						"id": "text1760183548",
						"max": 0,
						"min": 0,
						"name": "remote_url",
						"pattern": "",
						"presentable": false,
						"primaryKey": false,
						"required": true,
						"system": false,
						"type": "text"
					},
					{
						"autogeneratePattern": "",
						"help": "",
						"hidden": false,
						"id": "text4166911607",
						"max": 0,
						"min": 0,
						"name": "username",
						"pattern": "",
						"presentable": false,
						"primaryKey": false,
						"required": true,
						"system": false,
						"type": "text"
					},
					{
						"autogeneratePattern": "",
						"help": "",
						"hidden": false,
						"id": "text347813835",
						"max": 0,
						"min": 0,
						"name": "token_ciphertext",
						"pattern": "",
						"presentable": false,
						"primaryKey": false,
						"required": true,
						"system": false,
						"type": "text"
					},
					{
						"autogeneratePattern": "",
						"help": "",
						"hidden": false,
						"id": "text927719900",
						"max": 0,
						"min": 0,
						"name": "local_path",
						"pattern": "",
						"presentable": false,
						"primaryKey": false,
						"required": false,
						"system": false,
						"type": "text"
					},
					{
						"help": "",
						"hidden": false,
						"id": "bool1358543748",
						"name": "enabled",
						"presentable": false,
						"required": false,
						"system": false,
						"type": "bool"
					},
					{
						"autogeneratePattern": "",
						"help": "",
						"hidden": false,
						"id": "text2011621992",
						"max": 0,
						"min": 0,
						"name": "last_synced_at",
						"pattern": "",
						"presentable": false,
						"primaryKey": false,
						"required": false,
						"system": false,
						"type": "text"
					},
					{
						"autogeneratePattern": "",
						"help": "",
						"hidden": false,
						"id": "text1066830442",
						"max": 0,
						"min": 0,
						"name": "last_error",
						"pattern": "",
						"presentable": false,
						"primaryKey": false,
						"required": false,
						"system": false,
						"type": "text"
					},
					{
						"hidden": false,
						"id": "autodate2990389176",
						"name": "created",
						"onCreate": true,
						"onUpdate": false,
						"presentable": false,
						"system": false,
						"type": "autodate"
					},
					{
						"hidden": false,
						"id": "autodate3332085495",
						"name": "updated",
						"onCreate": true,
						"onUpdate": true,
						"presentable": false,
						"system": false,
						"type": "autodate"
					}
				],
				"id": "pbc_1325916001",
				"indexes": [],
				"listRule": null,
				"name": "connections",
				"system": false,
				"type": "base",
				"updateRule": null,
				"viewRule": null
			}
		]`

		return app.ImportCollectionsByMarshaledJSON([]byte(jsonData), false)
	}, func(app core.App) error {
		return nil
	})
}
