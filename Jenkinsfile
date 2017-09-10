node {

    stage 'Load shared libraries'

    def openshift, git
    def version='feature/gobygging'
    fileLoader.withGit('https://git.aurora.skead.no/scm/ao/aurora-pipeline-scripts.git', version) {
        openshift = fileLoader.load('openshift/openshift')
        git = fileLoader.load('git/git')
        go = fileLoader.load('go/go')
    }

    stage 'Checkout'
    checkout scm


    stage 'Test og coverage'
    go.buildGoWithJenkinsSh()

    stage 'OpenShift build'
    def commitId = git.getCommitId()
    def namespace = openshift.jenkinsNamespace()
    def result = openshift.oc("start-build architect --commit=${commitId} -n=${namespace} -F")
    if(!result) {
        error("Building docker image failed")
    }

}


