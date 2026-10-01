const cp = require('child_process')
const os = require('os')
const process = require('process')

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

function getAuthHeader() {
    const token = process.env.INPUT_TOKEN || process.env.GITHUB_TOKEN;
    if (token) {
        return `-H "Authorization: Bearer ${token}"`;
    }
    return '';
}

function getExecutablePath() {
    const isWindows = os.platform() === 'win32' || os.platform() === 'windows';
    return isWindows ? `.\\${BinaryName}.exe` : `./${BinaryName}`;
}

function determineVersion() {
    if (!!process.env.INPUT_VERSION) {
        return process.env.INPUT_VERSION
    }
    try {
        const authHeader = getAuthHeader();
        const headerArg = authHeader ? `${authHeader} ` : '';
        const raw = cp.execSync(`curl --silent --location ${headerArg}"https://api.github.com/repos/${OwnerRepo}/releases/latest"`).toString();
        const json = JSON.parse(raw);
        if (json && json.tag_name) {
            return json.tag_name;
        }
    } catch (e) {
        logDebug("Failed to determine latest version:", e);
    }
    logDebug("No version found, falling back to v1.3.1")
    return "v1.3.1"
}

function main() {
    logDebug(process.env.INPUT_REQUIRED_WORKFLOW_PATTERNS)
    logDebug("started")
    const versionTag = determineVersion()
    let status = downloadBinary(versionTag, chooseBinary(versionTag))
    if (typeof status === 'number' && status > 0) {
        process.exit(status)
    }

    console.log(`░▒▓███████▓▒░░▒▓████████▓▒░▒▓██████▓▒░░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░▒▓███████▓▒░░▒▓████████▓▒░▒▓███████▓▒░  
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

    const execPath = getExecutablePath();
    const spawnSyncReturns = cp.spawnSync(execPath, { stdio: 'inherit' })
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
