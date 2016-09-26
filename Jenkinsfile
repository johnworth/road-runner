#!groovy
node('docker') {
    slackJobDescription = "job '${env.JOB_NAME} [${env.BUILD_NUMBER}]' (${env.BUILD_URL})"
    try {
        stage "Build"
        checkout scm

        service = readProperties file: 'service.properties'

        git_commit = sh(returnStdout: true, script: "git rev-parse HEAD").trim()
        echo git_commit

        dockerRepo = "test-${env.BUILD_TAG}"

        sh "docker build --rm --build-arg git_commit=${git_commit} -t ${dockerRepo} ."


        dockerTestRunner = "test-${env.BUILD_TAG}"
        dockerTestCleanup = "test-cleanup-${env.BUILD_TAG}"
        try {
            stage "Test"
            try {
              sh "docker run --rm --name ${dockerTestRunner} --entrypoint 'sh' ${dockerRepo} -c \"go test -v github.com/cyverse-de/${service.repo} | tee /dev/stderr | go-junit-report\" > test-results.xml"
            } finally {
                junit 'test-results.xml'

                sh "docker run --rm --name ${dockerTestCleanup} -v \$(pwd):/build -w /build alpine rm -r test-results.xml"
            }


            stage "Docker Push"
            dockerPushRepo = "${service.dockerUser}/${service.repo}:${env.BRANCH_NAME}"
            sh "docker tag ${dockerRepo} ${dockerPushRepo}"
            sh "docker push ${dockerPushRepo}"
        } finally {
            sh returnStatus: true, script: "docker kill ${dockerTestRunner}"
            sh returnStatus: true, script: "docker rm ${dockerTestRunner}"

            sh returnStatus: true, script: "docker kill ${dockerTestCleanup}"
            sh returnStatus: true, script: "docker rm ${dockerTestCleanup}"

            sh returnStatus: true, script: "docker rmi ${dockerRepo}"
        }
    } catch (InterruptedException e) {
        currentBuild.result = "ABORTED"
        slackSend color: 'warning', message: "ABORTED: ${slackJobDescription}"
        throw e
    } catch (e) {
        currentBuild.result = "FAILED"
        sh "echo ${e}"
        slackSend color: 'danger', message: "FAILED: ${slackJobDescription}"
        throw e
    }
}