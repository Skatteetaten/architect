node {

    stage 'Load shared libraries'

    def openshift, git
    def scriptVersion='v5.5'
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

    stage('Deploy to Nexus'){
        def isMaster = env.BRANCH_NAME == 'master'

        def REPO_ID = isMaster ? 'releases' : 'snapshots'
        def REPO_URL = 'https://aurora/nexus/content/repositories/' + REPO_ID

        def version = git.getTagFromCommit()

        if (isMaster){
            if (!git.tagExists("v${version}")) {
                error "Commit is not tagged. Aborting build."
            }
        }

        def deployOpts = '-Durl=' + REPO_URL +
            ' -DrepositoryId=' + REPO_ID +
            ' -DgroupId=ske.aurora.openshift -DartifactId=architect -Dversion=' + version +
            ' -Dpackaging=tar.gz -DgeneratePom=true -Dfile=bin/amd64/architect.tar.gz'

        maven.setMavenVersion('Maven 3')
        maven.run('deploy:deploy-file', deployOpts)

    }

    stage 'OpenShift build'
    def commitId = git.getCommitId()
    def namespace = openshift.jenkinsNamespace()
    def result = openshift.oc("start-build architect --commit=${commitId} -n=${namespace} -F")
    if(!result) {
        error("Building docker image failed")
    }

}


