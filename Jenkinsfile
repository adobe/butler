node {

    def name = "Techops-iao/butler"
    def registry = 'http://docker-ethos-core-univ-release.dr-uw2.adobeitc.com'
    def repo = 'git@git.corp.adobe.com:TechOps-IAO/butler.git'
    def image
    def released
    def version

    stage('prepare') {
        checkout scm

        if (env.BRANCH_NAME == 'master') {
            version = readFile('VERSION').trim()
        } else {
            // See https://issues.jenkins-ci.org/browse/JENKINS-26100
            sh "git rev-parse --short HEAD > .git/commit-id"
            commit_id = readFile('.git/commit-id').trim()
            version = readFile('VERSION').trim() + '-' + commit_id
        }
    }

    stage('build') {
        withCredentials([usernamePassword(credentialsId: 'docker-registry-artifactory-aws', passwordVariable: 'PASSWORD', usernameVariable: 'USER')]) {
            sh("docker login -u $USER -p $PASSWORD ${registry}")
        }

        try {
            image = docker.image("${name}:${version}")
            docker.withRegistry(registry, 'docker-registry-artifactory-aws') {
                image.pull()
            }
            released = true
        } catch(exc) {
            sh "make build"
            image = docker.build("${name}:${version}", '.')

            released = false
        }
    }

    stage('test') {
        sh "make test"
    }

    if (env.BRANCH_NAME == 'master') {
        if (!released) {
            stage('release') {
                docker.withRegistry(registry, 'docker-registry-artifactory-aws') {
                    image.push()
                }

                sh "git tag -a ${version} -m 'Release ${version}'"
                sshagent(credentials: ['comj_git']) {
                    sh "git push ${repo} --tags"
                }
            }
        }
    } else {
        stage('release-dev') {
            registry = 'http://docker-ethos-core-univ-dev.dr-uw2.adobeitc.com'
            docker.withRegistry(registry) {
                image.push()
            }
        }
    }
}
