# Install or update wimamp on Windows from GitHub releases:
#   irm https://raw.githubusercontent.com/wimcraft/wimamp/main/install.ps1 | iex
# Options (env): WIMAMP_VERSION=v0.1.0 (default: latest),
#   WIMAMP_INSTALL_DIR (default: %LOCALAPPDATA%\Programs\wimamp)
# Works in Windows PowerShell 5.1 and PowerShell 7+. Everything runs inside a function, because a
# script piped to iex shares the caller's session: no leaked variables, and errors throw rather
# than `exit`, which would close the user's window.

function Install-Wimamp {
	Set-StrictMode -Version 2
	$ErrorActionPreference = 'Stop'
	$ProgressPreference = 'SilentlyContinue' # the progress bar makes Invoke-WebRequest many times slower in 5.1

	$repo = 'wimcraft/wimamp'
	$dir = if ($env:WIMAMP_INSTALL_DIR) { $env:WIMAMP_INSTALL_DIR } else { Join-Path $env:LOCALAPPDATA 'Programs\wimamp' }
	$version = if ($env:WIMAMP_VERSION) { $env:WIMAMP_VERSION } else { 'latest' }

	# PROCESSOR_ARCHITEW6432 is set when a 32-bit PowerShell runs on 64-bit Windows
	$cpu = if ($env:PROCESSOR_ARCHITEW6432) { $env:PROCESSOR_ARCHITEW6432 } else { $env:PROCESSOR_ARCHITECTURE }
	switch ($cpu) {
		'AMD64' { $arch = 'amd64' }
		'ARM64' { $arch = 'arm64' }
		default { throw "wimamp: unsupported CPU '$cpu'" }
	}

	$asset = "wimamp-windows-$arch.exe"
	if ($version -eq 'latest') {
		$base = "https://github.com/$repo/releases/latest/download"
	} else {
		$base = "https://github.com/$repo/releases/download/$version"
	}

	# Windows PowerShell 5.1 may default to TLS 1.0, which GitHub refuses
	[Net.ServicePointManager]::SecurityProtocol = [Net.ServicePointManager]::SecurityProtocol -bor [Net.SecurityProtocolType]::Tls12

	$tmp = Join-Path ([IO.Path]::GetTempPath()) ("wimamp-" + [Guid]::NewGuid())
	New-Item -ItemType Directory -Path $tmp | Out-Null
	try {
		Write-Host "Downloading $asset ($version)..."
		$exe = Join-Path $tmp $asset
		try {
			Invoke-WebRequest -UseBasicParsing -Uri "$base/$asset" -OutFile $exe
			Invoke-WebRequest -UseBasicParsing -Uri "$base/checksums.txt" -OutFile (Join-Path $tmp 'checksums.txt')
		} catch {
			throw "wimamp: could not download $asset ($version) from $base. Check the version at https://github.com/$repo/releases ($($_.Exception.Message))"
		}

		$expected = $null
		foreach ($line in Get-Content (Join-Path $tmp 'checksums.txt')) {
			$f = -split $line
			if ($f.Count -ge 2 -and $f[1].TrimStart('*') -eq $asset) { $expected = $f[0] }
		}
		$actual = (Get-FileHash -Algorithm SHA256 -Path $exe).Hash
		if (-not $expected -or $expected -ne $actual) { # -ne is case-insensitive; sha256sum prints lowercase
			throw "wimamp: checksum mismatch for $asset, not installing"
		}

		New-Item -ItemType Directory -Force -Path $dir | Out-Null
		$target = Join-Path $dir 'wimamp.exe'
		$old = "$target.old"
		Remove-Item -Force -ErrorAction SilentlyContinue $old # left behind by an update while wimamp was running
		# A running exe can't be overwritten, but it can be renamed: move it aside, then put the new one in place.
		if (Test-Path $target) { Move-Item -Force $target $old }
		try {
			Move-Item -Force $exe $target
		} catch {
			if (Test-Path $old) { Move-Item -Force $old $target }
			throw
		}
		Remove-Item -Force -ErrorAction SilentlyContinue $old # fails while the old wimamp still runs; the next update cleans up
	} finally {
		Remove-Item -Recurse -Force -ErrorAction SilentlyContinue $tmp
	}

	$installed = & $target version
	Write-Host "Installed $installed to $target"

	# Put the install dir on the user PATH (persisted) and on this session's PATH.
	$userPath = [Environment]::GetEnvironmentVariable('Path', 'User')
	$entries = @(if ($userPath) { $userPath -split ';' | Where-Object { $_ } })
	if ($entries -notcontains $dir -and $entries -notcontains "$dir\") {
		[Environment]::SetEnvironmentVariable('Path', (@($entries) + $dir) -join ';', 'User')
		Write-Host "Added $dir to your user PATH (other open terminals need a restart to see it)."
	}
	if (($env:Path -split ';') -notcontains $dir) { $env:Path = "$env:Path;$dir" }

	if (-not (Get-Command mpv -ErrorAction SilentlyContinue)) {
		# this session's PATH may predate an mpv install, so also look at the saved one
		$saved = ([Environment]::GetEnvironmentVariable('Path', 'Machine') + ';' + [Environment]::GetEnvironmentVariable('Path', 'User')) -split ';' |
			Where-Object { $_ -and (Test-Path -LiteralPath (Join-Path ([Environment]::ExpandEnvironmentVariables($_)) 'mpv.exe')) }
		$winget = Join-Path $env:ProgramFiles 'MPV Player' # where `winget install shinchiro.mpv` puts it, without touching PATH
		if ($saved) {
			Write-Host 'mpv is on your PATH, but this terminal started before it was added: open a new one.'
		} elseif (Test-Path (Join-Path $winget 'mpv.exe')) {
			Write-Host "mpv is installed in $winget but not on your PATH. Add it with:"
			Write-Host "  [Environment]::SetEnvironmentVariable('Path', [Environment]::GetEnvironmentVariable('Path', 'User') + ';$winget', 'User')"
		} else {
			Write-Host 'wimamp plays through mpv, which is not installed yet:'
			Write-Host '  scoop install mpv'
			Write-Host "  (or: winget install -e --id shinchiro.mpv, then add $winget to your PATH)"
		}
	}
}

Install-Wimamp
