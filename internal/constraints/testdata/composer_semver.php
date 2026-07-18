<?php

declare(strict_types=1);

if ($argc !== 2) {
    fwrite(STDERR, "Usage: composer_semver.php /path/to/composer\n");
    exit(2);
}

try {
    Phar::loadPhar($argv[1], 'composer.phar');
    require 'phar://composer.phar/vendor/autoload.php';

    $input = stream_get_contents(STDIN);
    $request = json_decode($input, true, 512, JSON_THROW_ON_ERROR);
    $parser = new Composer\Semver\VersionParser();
    $response = [
        'normalizations' => [],
        'matches' => [],
        'intersections' => [],
    ];

    foreach ($request['normalizations'] as $case) {
        try {
            $response['normalizations'][] = [
                'result' => true,
                'normalized' => $parser->normalize($case['version']),
            ];
        } catch (Throwable $error) {
            $response['normalizations'][] = [
                'result' => false,
                'error' => $error->getMessage(),
            ];
        }
    }

    foreach ($request['matches'] as $case) {
        try {
            $response['matches'][] = [
                'result' => Composer\Semver\Semver::satisfies(
                    $case['version'],
                    $case['constraint'],
                ),
            ];
        } catch (Throwable $error) {
            $response['matches'][] = ['result' => false, 'error' => $error->getMessage()];
        }
    }

    foreach ($request['intersections'] as $case) {
        try {
            $left = $parser->parseConstraints($case['left']);
            $right = $parser->parseConstraints($case['right']);
            $response['intersections'][] = [
                'result' => Composer\Semver\Intervals::haveIntersections($left, $right),
            ];
        } catch (Throwable $error) {
            $response['intersections'][] = [
                'result' => false,
                'error' => $error->getMessage(),
            ];
        }
    }

    echo json_encode($response, JSON_THROW_ON_ERROR);
} catch (Throwable $error) {
    fwrite(STDERR, $error->getMessage() . PHP_EOL);
    exit(1);
}
