<?php

namespace Ptp\Command;

use Composer\Command\BaseCommand;
use Symfony\Component\Console\Input\InputArgument;
use Symfony\Component\Console\Input\InputInterface;
use Symfony\Component\Console\Output\OutputInterface;
use Symfony\Component\Process\Process;

final class PtpCommand extends BaseCommand
{
    protected function configure(): void
    {
        $this
            ->setName('ptp')
            ->setDescription('Run PTP (PHP Test Processor) - parallel PHPUnit runner')
            ->setHelp('Run ptp subcommands, e.g. <info>composer ptp run</info> or <info>composer ptp list</info>')
            ->addArgument('ptp-args', InputArgument::IS_ARRAY, 'Arguments to pass to ptp (e.g. run, list, run --processors 8)', []);
    }

    protected function execute(InputInterface $input, OutputInterface $output): int
    {
        $vendorDir = $this->getComposer()->getConfig()->get('vendor-dir');
        $binary    = $vendorDir . \DIRECTORY_SEPARATOR . 'giorgi-leladze' . \DIRECTORY_SEPARATOR . 'ptp' . \DIRECTORY_SEPARATOR . 'bin' . \DIRECTORY_SEPARATOR . 'ptp-binary';

        if (!is_file($binary) || !is_executable($binary)) {
            $output->getErrorOutput()->writeln('<error>ptp binary not found. Run: composer update</error>');
            return 1;
        }

        $args = $input->getArgument('ptp-args');
        $proc = new Process(array_merge([$binary], $args));
        $proc->setTimeout(null);
        $proc->setTty(Process::isTtySupported());
        $proc->run(function (string $type, string $data): void {
            echo $data;
        });

        return $proc->getExitCode();
    }
}
