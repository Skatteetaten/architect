#!/usr/bin/env groovy
timestamps {
  node {

    def config = [
        artifactId               : 'architect',
        groupId                  : 'ske.aurora.openshift',
        deliveryBundleClassifier : "Doozerleveransepakke",
        scriptVersion            : 'v7',
        pipelineScript           : 'https://git.aurora.skead.no/scm/ao/aurora-pipeline-scripts.git',
        iqOrganizationName       : 'Team AOT',
        openShiftBaseImage       : 'builder',
        openShiftBaseImageVersion: 'latest',
        openShiftBuilderImage    : 'architect',
        openShiftBuilderVersion  : '1',
        versionStrategy          : [
            [branch: 'master', versionHint: '1']
        ],
        debug                    : true
    ]

    def tagVersion

    def maven
    def git
    def go
    def utilities
    def doozerleveranse

    stage('Load shared libraries') {
      fileLoader.withGit(config.pipelineScript, config.scriptVersion) {
        maven = fileLoader.load('maven/maven')
        git = fileLoader.load('git/git')
        go = fileLoader.load('go/go')
        utilities = fileLoader.load('utilities/utilities')
        doozerleveranse = fileLoader.load('templates/doozerleveranse')
      }
    }

    stage('Checkout') {
      checkout scm
    }

    stage('Test and coverage') {
      go.buildGoWithJenkinsSh("Go 1.12")
    }

    stage('Sonar') {
      def sonarPath = tool 'Sonar 4'
      sh "${sonarPath}/bin/sonar-scanner -Dsonar.branch.name=${env.BRANCH_NAME}"
    }

    stage('Deploy to Nexus') {
      def isMaster = env.BRANCH_NAME == 'master'
      tagVersion = git.executeAuroraGitVersionCliCommand(" --suggest-releases master --version-hint 1",
          utilities.getNexusInformation())

      if (isMaster) {
        git.tagIfNotExists('github', tagVersion)
      }

      maven.deployZipToNexusWithGroupId("bin/amd64/", config.artifactId, config.groupId, tagVersion, config.deliveryBundleClassifier)
    }

    doozerleveranse.run(config.scriptVersion, config)
  }
}
