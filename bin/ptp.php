<?php

/**
 * PTP launcher - runs the PTP binary installed by Composer.
 */

$binDir   = __DIR__;
$binary   = $binDir . DIRECTORY_SEPARATOR . 'ptp-binary';
$args     = array_slice($argv, 1);
$argList   = count($args) > 0 ? ' ' . implode(' ', array_map('escape_arg', $args)) : '';

if (!file_exists($binary) || !is_executable($binary)) {
    fwrite(STDERR, "ptp: binary not found. Run: composer update\n");
    exit(1);
}

$status = null;
passthru($binary . $argList, $status);
exit($status !== null ? $status : 1);

function escape_arg($arg)
{
    if ($arg === '' || preg_match('/[^\w\-.\/:=@]/', $arg)) {
        return "'" . str_replace("'", "'\\''", $arg) . "'";
    }
    return $arg;
}
