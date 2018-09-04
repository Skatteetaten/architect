node {

    stage 'Load shared libraries'

    def openshift, git
    def scriptVersion='v5'
    fileLoader.withGit('https://git.aurora.skead.no/scm/ao/aurora-pipeline-scripts.git', scriptVersion) {
        openshift = fileLoader.load('openshift/openshift')
        maven = fileLoader.load('maven/maven')
        git = fileLoader.load('git/git')
        go = fileLoader.load('go/go')
    }

    stage 'Checkout'
    checkout scm


    stage 'Test og coverage'
    go.buildGoWithJenkinsSh()

    stage 'Deploy to Nexus'
    def isMaster = env.BRANCH_NAME == 'master'
    def tagVersion = git.getTagFromCommit()

    if (isMaster){
        if (!git.tagExists("v${tagVersion}")) {
            error "Commit is not tagged. Aborting build."
        }
    }
    maven.deployTarGzToNexus("bin/amd64/", "architect", tagVersion)
    

    stage 'OpenShift build'
    def namespace = openshift.jenkinsNamespace()
    def result = openshift.oc("start-build architect -e ARTIFACT_NAME=architect -e ARTIFACT_VERSION=${tagVersion} -n=${namespace} -F")
    if(!result) {
        error("Building docker image failed")
    }

}


