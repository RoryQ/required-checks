const cp = require('child_process')
const os = require('os')
const process = require('process')

const DefaultVersionFallback = "v1.2.0"
const OwnerRepo = "RoryQ/required-checks"
const BinaryName = "required-checks"

function logDebug(...data) {
    if (!!process.env.REQUIRED_CHECKS_DEBUG) {
        console.debug(...data)
    }
}

function chooseBinary(versionTag) {
    const platform = os.platform()
    const arch = os.arch()

    const binaryVersion = versionTag.substring(1);

    if (platform === 'linux' && arch === 'x64') {
        return `${BinaryName}_${binaryVersion}_Linux_x86_64`
    }
    if (platform === 'linux' && arch === 'arm64') {
        return `${BinaryName}_${binaryVersion}_Linux_arm64`
    }
    if (platform === 'windows' && arch === 'x64') {
        return `${BinaryName}_${binaryVersion}_Windows_x86_64`
    }
    if (platform === 'windows' && arch === 'arm64') {
        return `${BinaryName}_${binaryVersion}_Windows_arm64`
    }
    if (platform === 'darwin' && arch === 'x64') {
        return `${BinaryName}_${binaryVersion}_Darwin_x86_64`
    }
    if (platform === 'darwin' && arch === 'arm64') {
        return `${BinaryName}_${binaryVersion}_Darwin_arm64`
    }

    console.error(`Unsupported platform (${platform}) and architecture (${arch})`)
    process.exit(1)
}

function downloadBinary(versionTag, binary) {
    const url = `https://github.com/${OwnerRepo}/releases/download/${versionTag}/${binary}.tar.gz`
    logDebug(url)
    const result = cp.execSync(`curl --silent --location --remote-header-name  ${url} | tar xvz`)
    logDebug(result.toString())
    return result.status
}

function determineVersion() {
    if (!!process.env.INPUT_VERSION) {
        return process.env.INPUT_VERSION
    }
    const result = cp.execSync(`curl --silent --location "https://api.github.com/repos/${OwnerRepo}/releases/latest" | jq  -r ".. .tag_name? // empty"`)
    let ver = result.toString().trim();
    // fallback to the default version
    if (ver === "") {
        logDebug("No version found, falling back to "+DefaultVersionFallback)
        ver = DefaultVersionFallback
    }
    return ver
}

function main() {
    logDebug(process.env.INPUT_REQUIRED_WORKFLOW_PATTERNS)
    logDebug("started")
    const versionTag = determineVersion()
    let status = downloadBinary(versionTag, chooseBinary(versionTag))
    if (typeof status === 'number' && status > 0) {
        process.exit(status)
    }

    console.log(
`░▒▓███████▓▒░░▒▓████████▓▒░▒▓██████▓▒░░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░▒▓███████▓▒░░▒▓████████▓▒░▒▓███████▓▒░  
░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░     ░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░      ░▒▓█▓▒░░▒▓█▓▒░ 
░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░     ░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░      ░▒▓█▓▒░░▒▓█▓▒░ 
░▒▓███████▓▒░░▒▓██████▓▒░░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░▒▓███████▓▒░░▒▓██████▓▒░ ░▒▓█▓▒░░▒▓█▓▒░ 
░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░     ░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░      ░▒▓█▓▒░░▒▓█▓▒░ 
░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░     ░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░      ░▒▓█▓▒░░▒▓█▓▒░ 
░▒▓█▓▒░░▒▓█▓▒░▒▓████████▓▒░▒▓██████▓▒░ ░▒▓██████▓▒░░▒▓█▓▒░▒▓█▓▒░░▒▓█▓▒░▒▓████████▓▒░▒▓███████▓▒░  
                            ░▒▓█▓▒░                                                               
                             ░▒▓██▓▒░                                                             
 ░▒▓██████▓▒░░▒▓█▓▒░░▒▓█▓▒░▒▓████████▓▒░▒▓██████▓▒░░▒▓█▓▒░░▒▓█▓▒░░▒▓███████▓▒░                    
░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░     ░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░                           
░▒▓█▓▒░      ░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░     ░▒▓█▓▒░      ░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░                           
░▒▓█▓▒░      ░▒▓████████▓▒░▒▓██████▓▒░░▒▓█▓▒░      ░▒▓███████▓▒░ ░▒▓██████▓▒░                     
░▒▓█▓▒░      ░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░     ░▒▓█▓▒░      ░▒▓█▓▒░░▒▓█▓▒░      ░▒▓█▓▒░                    
░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░     ░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░░▒▓█▓▒░      ░▒▓█▓▒░                    
 ░▒▓██████▓▒░░▒▓█▓▒░░▒▓█▓▒░▒▓████████▓▒░▒▓██████▓▒░░▒▓█▓▒░░▒▓█▓▒░▒▓███████▓▒░                     
                                                                                                  
                                                                                                  `)

    const spawnSyncReturns = cp.spawnSync(`./${BinaryName}`, { stdio: 'inherit' })
    logDebug(spawnSyncReturns)
    status = spawnSyncReturns.status
    if (typeof status === 'number') {
        process.exit(status)
    }
    process.exit(1)
}

if (require.main === module) {
    main()
}
