#!/usr/bin/env php
<?php

/**
 * Downloads the PTP binary for the current OS/arch from GitHub Releases.
 * Runs as Composer post-install/post-update.
 */

$packageRoot = dirname(__DIR__);
$binDir     = $packageRoot . DIRECTORY_SEPARATOR . 'bin';
$binaryPath = $binDir . DIRECTORY_SEPARATOR . 'ptp-binary';

$asset = getAssetName();
if ($asset === null) {
    fprintf(STDERR, "ptp: unsupported platform (%s %s). Install manually from https://github.com/giorgi-leladze/php-test-processor/releases\n", PHP_OS_FAMILY, php_uname('m'));
    exit(1);
}

$url = 'https://github.com/giorgi-leladze/php-test-processor/releases/latest/download/' . $asset . '.tar.gz';

if (!is_dir($binDir)) {
    mkdir($binDir, 0755, true);
}

$tmpFile = tempnam(sys_get_temp_dir(), 'ptp-') . '.tar.gz';

$context = stream_context_create([
    'http' => [
        'follow_location' => 1,
        'user_agent'      => 'ptp-composer-installer/1.0',
    ],
]);

$data = @file_get_contents($url, false, $context);
if ($data === false) {
    fprintf(STDERR, "ptp: failed to download %s\n", $url);
    exit(1);
}

file_put_contents($tmpFile, $data);

$extractDir = sys_get_temp_dir() . DIRECTORY_SEPARATOR . 'ptp-extract-' . getmypid();
mkdir($extractDir, 0755, true);

// Use tar to extract (no phar extension required)
$cmd = sprintf('tar -xzf %s -C %s 2>/dev/null', escapeshellarg($tmpFile), escapeshellarg($extractDir));
exec($cmd, $out, $code);
unlink($tmpFile);
if ($code !== 0) {
    fprintf(STDERR, "ptp: failed to extract archive\n");
    exit(1);
}

$extractedName = $asset;
$extractedPath = $extractDir . DIRECTORY_SEPARATOR . $extractedName;
if (!file_exists($extractedPath)) {
    fprintf(STDERR, "ptp: extracted binary not found at %s\n", $extractedPath);
    exit(1);
}

if (file_exists($binaryPath)) {
    @unlink($binaryPath);
}
rename($extractedPath, $binaryPath);
chmod($binaryPath, 0755);

// Cleanup temp extract dir (file was moved out, so dir is empty)
@rmdir($extractDir);

/**
 * @return string|null e.g. "ptp-linux-amd64" or null if unsupported
 */
function getAssetName(): ?string
{
    $os   = PHP_OS_FAMILY;
    $arch = strtolower(php_uname('m'));

    $archMap = [
        'x86_64' => 'amd64',
        'amd64'  => 'amd64',
        'aarch64' => 'arm64',
        'arm64'  => 'arm64',
    ];
    $arch = $archMap[$arch] ?? $arch;

    if ($os === 'Linux' && $arch === 'amd64') {
        return 'ptp-linux-amd64';
    }
    if ($os === 'Darwin' && $arch === 'amd64') {
        return 'ptp-darwin-amd64';
    }
    if ($os === 'Darwin' && $arch === 'arm64') {
        return 'ptp-darwin-arm64';
    }

    return null;
}
