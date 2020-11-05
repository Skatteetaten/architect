#!/usr/bin/env groovy


def goleveranse
def config = [
    artifactId               : 'architect',
    groupId                  : 'aurora',
    deliveryBundleClassifier : "Doozerleveransepakke",
    scriptVersion            : 'v7',
    pipelineScript           : 'https://git.aurora.skead.no/scm/ao/aurora-pipeline-scripts.git',
    iqOrganizationName       : 'Team AppSikk',
    openShiftBaseImage       : 'builder',
    openShiftBaseImageVersion: 'latest',
    goVersion                : '1.13',
    artifactPath             : 'bin/amd64/',
    chatRoom                 : "#sitj-build",
    credentialsId: "github",
    versionStrategy          : [
        [branch: 'master', versionHint: '1']
        ]
    ]

fileLoader.withGit(config.pipelineScript, config.scriptVersion) {
  goleveranse = fileLoader.load('templates/goleveranse')
}

goleveranse.run(config.scriptVersion, config)




