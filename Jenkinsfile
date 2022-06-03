#!/usr/bin/env groovy


def goleveranse
def config = [
    artifactId               : 'architect',
    groupId                  : 'aurora',
    deliveryBundleClassifier : "Doozerleveransepakke",
    scriptVersion            : 'v7',
    pipelineScript           : 'https://git.aurora.skead.no/scm/ao/aurora-pipeline-scripts.git',
    iqOrganizationName       : 'Team APS',
    openShiftBaseImage       : 'builder',
    openShiftBaseImageVersion: 'latest',
    goVersion                : '1.18',
    artifactPath             : 'bin/',
    chatRoom                 : "#sitj-build",
    credentialsId: "github",
    versionStrategy          : [
        [branch: 'master', versionHint: '2']
        ]
    ]

fileLoader.withGit(config.pipelineScript, config.scriptVersion) {
  goleveranse = fileLoader.load('templates/goleveranse')
}

goleveranse.run(config.scriptVersion, config)




