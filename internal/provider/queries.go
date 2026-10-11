package provider

type QueryGroups struct {
	queries    []string
	outputFile string
	searchType string
}

var AllQueryGroups = map[string]*QueryGroups{
	"cloud-endpoints": {
		queries:    CloudQueries[:],
		outputFile: "cloud_endpoints.json",
		searchType: "messages",
	},
	"credentials": {
		queries:    CredentialQueries[:],
		outputFile: "credentials.json",
		searchType: "messages",
	},
	"connection-strings": {
		queries:    ConnectionStringQueries[:],
		outputFile: "connection_strings.json",
		searchType: "messages",
	},
	"active-directory": {
		queries:    ActiveDirectoryQueries[:],
		outputFile: "active_directory.json",
		searchType: "messages",
	},
	"api-keys": {
		queries:    APIPrefixQueries[:],
		outputFile: "api_keys.json",
		searchType: "messages",
	},
	"private-keys": {
		queries:    PrivateKeyQueries[:],
		outputFile: "private_keys.json",
		searchType: "messages",
	},
	"interesting-files": {
		queries:    InterestingFileQueries[:],
		outputFile: "interesting_files.json",
		searchType: "files",
	},
}

var CloudQueries = [...]string{
	// AWS
	"s3.amazonaws.com",
	"elb.amazonaws.com",
	"rds.amazonaws.com",
	"amazon.com/s3/*",
	"*://s3*",
	"cloudfront.net",
	// Azure
	"core.windows.net",
	"azurewebsites.net",
	"vault.azure.net",
	"database.windows.net",
	// GCP
	"storage.googleapis.com",
	"storage.cloud.google.com",
	"gs://*",
	//Cloud storage
	"drive.google.com",
	"dropbox.com",
	"my.sharepoint.com",
	// misc
	"digitaloceanspaces.com",
	"cloudflarestorage.com",
	"backblazeb2.com",
}

const CloudRegex = `(?i)(` +
	// AWS
	`[a-zA-Z0-9\-\.\_]+\.s3[.-][a-z0-9\-]*\.amazonaws\.com` +
	`|s3://[a-zA-Z0-9\-\.\_]+` +
	`|s3\.[a-zA-Z0-9\-]+\.amazonaws\.com` +
	`|s3\.amazonaws\.com/[a-zA-Z0-9\-\.\_]+` +
	`|s3\.console\.aws\.amazon\.com/s3/buckets/[a-zA-Z0-9\-\.\_]+` +
	`|[a-zA-Z0-9\-]+\.execute-api\.[a-z0-9\-]+\.amazonaws\.com` +
	`|[a-zA-Z0-9\-]+\.cloudfront\.net` +
	`|[a-zA-Z0-9\-]+\.elb\.amazonaws\.com` +
	`|[a-zA-Z0-9\-]+\.rds\.amazonaws\.com` +
	// Azure
	`|[a-zA-Z0-9\-\.\_]+\.blob\.core\.windows\.net` +
	`|[a-zA-Z0-9\-\.\_]+\.file\.core\.windows\.net` +
	`|[a-zA-Z0-9\-\.\_]+\.dfs\.core\.windows\.net` +
	`|[a-zA-Z0-9\-\.\_]+\.queue\.core\.windows\.net` +
	`|[a-zA-Z0-9\-\.\_]+\.table\.core\.windows\.net` +
	`|[a-zA-Z0-9\-]+\.azurewebsites\.net` +
	`|[a-zA-Z0-9\-]+\.vault\.azure\.net` +
	`|[a-zA-Z0-9\-]+\.database\.windows\.net` +
	// GCP
	`|[a-zA-Z0-9\-_.]+\.storage\.googleapis\.com` +
	`|storage\.googleapis\.com/[a-zA-Z0-9\-_.]+` +
	`|storage\.cloud\.google\.com/[a-zA-Z0-9\-_./]+` +
	`|gs://[a-zA-Z0-9\-_.]+` +
	// Personal cloud storage
	`|drive\.google\.com/(?:file/d/|drive/folders/|open\?id=)[a-zA-Z0-9\-_]+` +
	`|[a-zA-Z0-9\-]+\.dropbox\.com/[a-zA-Z0-9\-_./?=&]+` +
	`|[a-zA-Z0-9\-]+-my\.sharepoint\.com/[a-zA-Z0-9\-_./:?=&]+` +
	// Other providers
	`|[a-zA-Z0-9\-]+\.digitaloceanspaces\.com` +
	`|[a-zA-Z0-9\-]+\.r2\.cloudflarestorage\.com` +
	`|[a-zA-Z0-9\-]+\.backblazeb2\.com` +
	`)`

var CredentialQueries = [...]string{
	"password*", "passwd*", "password*", "pass*", "pwd*",
	"login*", "creds*", "credentials*", "cred*",
	"api_key*", "api key*", "apikey*",
	"client_secret*", "client secret*",
	"token*", "access_token*", "access token*", "auth_token*", "bearer*",
	"passphrase*",
}

const CredentialRegex = `(?i)(` +
	`password\s*[\x60=:"]+\s*[^\s]+|` +
	`password is\s*[\x60=:"]*\s*[^\s]+|` +
	`pwd\s*[\x60=:"]*\s*[^\s]+|` +
	`passwd\s*[\x60=:"]+\s*[^\s]+)`

var ConnectionStringQueries = [...]string{
	"connection string", "conn_str", "connectionstring",
	"jdbc:", "mongodb://",
	"postgres://", "postgresql://",
	"mysql://",
	"redis://",
	"amqp://",
	"mssql://", "Server=", "sqlserver://",
	"oracle://",
	"cassandra://",
	"elasticsearch://",
	"ldap://", "ldaps://",
	"smtp://", "smtps://",
	"ftp://", "sftp://",
	"DefaultEndpointsProtocol=",
	"Data Source=", "User ID=",
}

var ActiveDirectoryQueries = [...]string{
	"svc_", "service account",
	"domain admin", // maybe information about who the DAs are
	"bind dn", "bindDN",
	"sAMAccountName",
	"userPrincipalName",
	"DC=", "OU=",
	"krbtgt", // gotta be spicy if they are mentioning krbtgt
	"domain\\",
}

var APIPrefixQueries = [...]string{
	// hopefully the secret keys will be shared as well
	"AKIA*",                  // the good stuff long-term access key ID
	"ASIA*",                  // temporary access key ID
	"AIDA*",                  // IAM user ID — sometimes leaked alongside more useful info
	"AROA*",                  // IAM role ID — sometimes leaked alongside more useful info
	"dop_v1_*",               // Personal Access Token
	"doo_v1_*",               // OAuth access token
	"AIza*",                  // GCP/Firebase API key prefix
	"sk-ant-api03-*",         // Anthropic API key
	"sk-proj-*",              // OpenAI project API key
	"sk-live-*", "rk_live_*", // Stripe secret/restricted keys
	"ghp_*", "gho_*", "ghu_*", "ghs_*", "ghr_*", // GitHub tokens (classic + fine-grained prefixes)
	"xox*",    // Slack tokens
	"glpat-*", // GitLab personal access token

	// too many false positives but uncomment if you want them
	// "npm_*",   // npm automation token
	// "SG.*",    // SendGrid API key
}

var PrivateKeyQueries = [...]string{"BEGIN DSA PRIVATE",
	"BEGIN EC PRIVATE",
	"BEGIN OPENSSH PRIVATE",
	"BEGIN PGP PRIVATE",
	"BEGIN RSA PRIVATE",
	"BEGIN PRIVATE KEY",
	"BEGIN ENCRYPTED PRIVATE KEY",
	"PuTTY-User-Key-File",
}

var InterestingFileQueries = [...]string{
	// key/cert/creds
	".config", ".key", ".p12", ".pem", ".pfx", ".pkcs12", ".ppk",
	".crt", ".cer", ".der", ".jks", ".keystore", ".asc",
	// sourcecode and scripts
	".sh", ".zsh", ".py", ".js", ".json", ".jsp", ".md", ".pl", ".rb",
	// data or configs
	".sql", ".xml", ".yaml", ".yml", ".toml", ".csv", ".xlsx", ".xls",
	// archives
	".tar", ".tar.gz", ".tgz", ".zip", ".7z", ".rar", ".bz2",
	// logs
	".txt", ".log",
	// databases
	".db", ".sqlite", ".sqlite3", ".db3", ".mdb", ".accdb", ".bson", ".rdb",
	// Terraform
	".tfstate", ".tfvars",
	// ssh
	"id_rsa", "id_ed",
	// env and secrets
	".env", ".env.local", ".env.production",
	".netrc", "_netrc",
	".npmrc", ".pypirc",
	".htpasswd",
	".git-credentials",
	".dockercfg", "docker/config.json",
	".kube/config", "kubeconfig",
	// useful named files
	"dockerfile", "docker-compose.yml", "docker-compose.yaml",
	"gitignore", "git-crypt", "gpg", "keychain", "kdbx",
	"ansible-vault", "vault",
	// extra interesting queries if they have files associated with them
	"backup", "config", "credentials", "db", "database",
	"dump", "env", "password", "secret",
}
