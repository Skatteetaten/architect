node {

    stage 'Load shared libraries'

    def openshift, git
    def version='v3.0.0'
    fileLoader.withGit('https://git.aurora.skead.no/scm/ao/aurora-pipeline-scripts.git', version) {
        openshift = fileLoader.load('openshift/openshift')
        git = fileLoader.load('git/git')
    }

    stage 'Checkout'
    checkout scm

    stage 'Bygg Architect'

    def commitId = git.getCommitId()
    def result = openshift.oc("start-build architect --commit=${commitId} -F")
    if(!result) {
       error("Building docker image failed")
    }
}


