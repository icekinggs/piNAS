# Security Policy

PiNAS is a self-hosted NAS project. Please report vulnerabilities privately before public disclosure.

## Reporting

Open a private security advisory on GitHub if available, or contact the maintainer through the repository owner profile.

Include:

- Affected version or commit SHA.
- Steps to reproduce.
- Impact and required privileges.
- Suggested fix, if known.

## Scope

High-priority areas:

- Authentication and session handling.
- Filesystem jail escapes.
- Samba user/share management.
- Installer commands that can overwrite host configuration.
- Exposure of secrets in logs, browser storage, or state files.
