#!/usr/bin/env groovy


def goleveranse
def config = [
    artifactId               : 'architect',
    groupId                  : 'ske.aurora.openshift',
    deliveryBundleClassifier : "Doozerleveransepakke",
    scriptVersion            : 'v7',
    pipelineScript           : 'https://git.aurora.skead.no/scm/ao/aurora-pipeline-scripts.git',
    iqOrganizationName       : 'Team AOT',
    openShiftBaseImage       : 'builder',
    openShiftBaseImageVersion: 'latest',
    goVersion                : 'Go 1.12',
    artifactPath             : 'bin/amd64/',
    credentialsId: "github",
    versionStrategy          : [
        [branch: 'master', versionHint: '1']
        ]
    ]

fileLoader.withGit(config.pipelineScript, config.scriptVersion) {
  goleveranse = fileLoader.load('templates/goleveranse')
}

goleveranse.run(config.scriptVersion, config)




