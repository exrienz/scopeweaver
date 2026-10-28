#!/usr/bin/env node
// Exercise message helpers and disposable command stubs, never a real install/reset.
import assert from 'node:assert/strict';
import { readFileSync, writeFileSync, mkdtempSync, mkdirSync, rmSync, existsSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { resolve, dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';
import { spawn, spawnSync } from 'node:child_process';
import { SUPPORTED_LANGUAGES } from '../adapters/agent/src/client.js';

const repo = resolve(dirname(fileURLToPath(import.meta.url)), '..');
const scripts = ['install.sh', 'build.sh', 'start.sh', 'update.sh', 'reset-password.sh', 'dev.sh'];
const languages = JSON.parse(readFileSync(join(repo, 'locale/definitions.json'))).locales;
assert.deepEqual([...SUPPORTED_LANGUAGES], languages.map(language => language.id), 'standalone adapter must agree with the shipped registry');
const vectors = JSON.parse(readFileSync(join(repo, 'locale/matching-vectors.json'))).vectors;
const env = { ...process.env, SCOPEWEAVER_LANGUAGE: '', ARTEX_LANGUAGE: '' };
const q = value => `'${value.replaceAll("'", "'\\''")}'`;
const text = name => readFileSync(join(repo, name), 'utf8');
const run = (shell, command, overrides = {}, options = {}) => spawnSync(shell, ['-c', command], {
  env: { ...env, ...overrides }, encoding: 'utf8', timeout: 10000, ...options,
});
const success = result => assert.equal(result.status, 0, result.stderr || result.stdout);
const quoted = String.raw`(?:'(?:[^']*)'|"(?:\\.|[^"\\])*")`;
const calls = new RegExp(String.raw`\bmsg\s+(${quoted}(?:\s+${quoted}){5})`, 'g');
let messageCount = 0;

for (const name of scripts) {
  const source = text(name);
  const start = source.indexOf('normalize_language()');
  assert.ok(start >= 0, `${name}: missing language helper`);
  const end = source.indexOf('\n}', source.indexOf('msg() {', start)) + 2;
  const helper = source.slice(start, end);
  const shell = name === 'start.sh' ? 'sh' : 'bash';
  for (const { input, expected } of vectors) {
    const result = run(shell, `${helper}\nprintf '%s' "$LANG_SEL"`, { SCOPEWEAVER_LANGUAGE: input });
    success(result);
    assert.equal(result.stdout, expected || 'en', `${name}: ${input}`);
  }
  for (const language of languages) for (const alias of language.aliases) {
    const result = run(shell, `${helper}\nprintf '%s' "$LANG_SEL"`, { ARTEX_LANGUAGE: alias });
    success(result);
    assert.equal(result.stdout, language.id, `${name}: legacy variable ${alias}`);
  }
  for (const [modern, legacy, expected] of [['ja', 'ko', 'ja'], ['unsupported', 'ko', 'en'], ['', 'es-MX', 'es']]) {
    const result = run(shell, `${helper}\nprintf '%s' "$LANG_SEL"`, { SCOPEWEAVER_LANGUAGE: modern, ARTEX_LANGUAGE: legacy });
    success(result);
    assert.equal(result.stdout, expected, `${name}: environment precedence`);
  }
  const matches = [...source.matchAll(calls)];
  const callStarts = [...source.matchAll(/\$\(msg\s/g)];
  assert.equal(matches.length, callStarts.length, `${name}: every message needs six arguments`);
  assert.ok(matches.length > 0, `${name}: no message calls found`);
  for (const match of matches) {
    const args = match[1].match(new RegExp(quoted, 'g'));
    assert.equal(args.length, 6);
    const variables = value => [...value.matchAll(/\$(?:\{[^}]+\}|[A-Za-z_][A-Za-z_0-9]*|[1-9*])/g)].map(m => m[0].replace(/[{}]/g, '')).sort();
    const placeholders = variables(args[0]);
    for (const arg of args) assert.deepEqual(variables(arg), placeholders, `${name}: ${args[0]}`);
    // Spanish uses the same word "Error"; identical wording here is intentional.
    for (let i = 2; i < 6; i++) if (!(i === 5 && args[0].slice(1, -1) === 'Error: $*')) assert.notEqual(args[i].slice(1, -1), args[0].slice(1, -1), `${name}: English fallback`);
    // Exercise actual interpolation with harmless fixture values and no application commands.
    for (const language of languages) {
      const values = Object.fromEntries(['BIN','binary','before','after','archive','target','goos','goarch','ARTEX_BUILD_VERSION','output','checksum_file','ARTEX_PACKAGE_DIR','code','delay','MODE','cfg','CONTAINER','DUSER','DNAME','EXEC_KIND'].map(key => [key, `fixture_${key}`]));
      const result = run(shell, `set -u\nset -- fixture_argument\n${helper}\nmsg ${match[1]}`, { ...values, SCOPEWEAVER_LANGUAGE: language.id });
      success(result);
      assert.ok(result.stdout.trim().length > 0, `${name}: empty message for ${language.id}`);
    }
    messageCount++;
  }
}
console.log(`PASS: six script helpers, matching vectors/aliases/precedence, ${messageCount} six-language messages and unchanged interpolation`);

const fixture = mkdtempSync(join(tmpdir(), 'scopeweaver-script-locales-'));
const executable = (name, body) => writeFileSync(join(fixture, name), body, { mode: 0o755 });
try {
  mkdirSync(join(fixture, 'bin'));
  executable('start.sh', text('start.sh'));
  executable('reset-password.sh', text('reset-password.sh'));
  executable('update.sh', text('update.sh'));
  executable('bin/sleep', '#!/bin/sh\nprintf "%s\\n" "$1" >> "$FIXTURE/sleeps"\n');
  executable('bin/psql', '#!/bin/sh\nprintf "%s\\n" "$@" > "$FIXTURE/psql-args"\nprintf "%s" "$ARTEX_RESET_NEWPASS" > "$FIXTURE/password"\ncat > "$FIXTURE/sql"\n');
  executable('bin/docker', '#!/bin/sh\nprintf "%s\\n" "$*" >> "$FIXTURE/docker"\n');
  const fixtureEnv = { ...env, FIXTURE: fixture, PATH: `${join(fixture, 'bin')}:${process.env.PATH}` };
  const execute = (name, args = [], input = '', language = 'en') => spawnSync(name === 'start.sh' ? 'sh' : 'bash', [join(fixture, name), ...args], {
    cwd: fixture, env: { ...fixtureEnv, SCOPEWEAVER_LANGUAGE: language }, input, encoding: 'utf8', timeout: 10000,
  });
  const clean = (...names) => { for (const name of names) rmSync(join(fixture, name), { force: true }); };
  const contents = name => readFileSync(join(fixture, name), 'utf8');
  const expectedSleeps = '1\n2\n4\n8\n16\n32\n60\n60\n1\n';
  for (const language of languages) {
    clean('scopeweaver', 'count', 'sleeps');
    const missing = execute('start.sh', [], '', language.id);
    assert.equal(missing.status, 1, missing.stderr);
    assert.ok(missing.stderr.includes('./scopeweaver'));
    executable('scopeweaver', `#!/bin/sh\nn=0\n[ ! -f "$FIXTURE/count" ] || n=$(cat "$FIXTURE/count")\nn=$((n+1))\nprintf '%s' "$n" > "$FIXTURE/count"\nprintf '%s\\n' "$@" > "$FIXTURE/argv"\ncase "$n" in 1|10) exit 75 ;; 2|3|4|5|6|7|8|9|11) exit 7 ;; *) exit 0 ;; esac\n`);
    const result = execute('start.sh', ['argument with spaces', '--literal=*'], '', language.id);
    success(result);
    assert.equal(contents('count'), '12');
    assert.equal(contents('sleeps'), expectedSleeps);
    assert.equal(contents('argv'), 'argument with spaces\n--literal=*\n');
    // The reset is executed only against a stub psql; no database is contacted.
    for (const answer of ['\n', 'n\n']) {
      clean('sql');
      const cancelled = execute('reset-password.sh', ['--dsn', 'fixture-only', '-p', 'fixture-password'], answer, language.id);
      assert.equal(cancelled.status, 1, cancelled.stderr);
      assert.ok(!existsSync(join(fixture, 'sql')), 'cancellation reached psql');
    }
    for (const answer of ['y\n', 'Y\n']) {
      const reset = execute('reset-password.sh', ['--dsn', 'fixture-only', '-p', 'fixture-password'], answer, language.id);
      success(reset);
      assert.equal(contents('password'), 'fixture-password');
      assert.ok(!contents('psql-args').includes('fixture-password'));
      assert.ok(contents('sql').includes("crypt(:'newpw', gen_salt('bf', 10))"));
      assert.ok(contents('sql').includes('\\getenv newpw ARTEX_RESET_NEWPASS'));
    }
    clean('sql');
    const mismatch = execute('reset-password.sh', ['--dsn', 'fixture-only'], 'first\nsecond\n', language.id);
    assert.equal(mismatch.status, 1, mismatch.stderr);
    assert.ok(!existsSync(join(fixture, 'sql')), 'password mismatch reached psql');
    const empty = execute('reset-password.sh', ['--dsn', 'fixture-only'], '\n', language.id);
    assert.equal(empty.status, 1, empty.stderr);
    assert.ok(!existsSync(join(fixture, 'sql')), 'empty password reached psql');
    writeFileSync(join(fixture, '.env'), 'POSTGRES_PASSWORD=fixture-existing\n');
    clean('docker');
    const updated = execute('update.sh', [], '\n', language.id);
    success(updated);
    assert.equal(contents('docker'), 'compose version\ncompose build scopeweaver\ncompose up -d scopeweaver\n');
    assert.equal(contents('.env'), 'POSTGRES_PASSWORD=fixture-existing\n');
    const invalid = execute('update.sh', [], 'invalid\n', language.id);
    assert.equal(invalid.status, 1, invalid.stderr);
    const help = execute('reset-password.sh', ['--help'], '', language.id);
    success(help);
    assert.ok(help.stdout.includes('-p/--new-password PASSWORD; -y/--yes; -h/--help'));
    const buildHelp = spawnSync('bash', [join(repo, 'build.sh'), '--help'], { env: { ...env, SCOPEWEAVER_LANGUAGE: language.id }, encoding: 'utf8' });
    success(buildHelp);
    for (const token of ['--target OS/ARCH', '--release', '--upx', '--no-compress', 'ARTEX_TARGETS=linux/amd64,windows/amd64 ./build.sh --release']) assert.ok(buildHelp.stdout.includes(token));
  }
  console.log('PASS: all-six supervisor exits/restarts/capped backoff/argument forwarding; stubbed reset confirmation/cancellation/password safety; update default/config preservation and help');

  clean('ready', 'forwarded');
  executable('scopeweaver', `#!/bin/sh\ntrap 'printf forwarded > "$FIXTURE/forwarded"; exit 0' TERM\nprintf ready > "$FIXTURE/ready"\nwhile :; do /bin/sleep 0.05; done\n`);
  const child = spawn('sh', [join(fixture, 'start.sh')], { cwd: fixture, env: { ...fixtureEnv, SCOPEWEAVER_LANGUAGE: 'en' }, stdio: ['ignore', 'pipe', 'pipe'] });
  let output = '';
  child.stdout.on('data', data => { output += data; });
  const completion = new Promise(resolve => child.on('exit', (code, signal) => resolve({ code, signal })));
  const deadline = Date.now() + 5000;
  while (!existsSync(join(fixture, 'ready')) && Date.now() < deadline) await new Promise(resolve => setTimeout(resolve, 20));
  assert.ok(existsSync(join(fixture, 'ready')), 'stub child did not become ready');
  child.kill('SIGTERM');
  const timer = setTimeout(() => child.kill('SIGKILL'), 5000);
  const stopped = await completion;
  clearTimeout(timer);
  assert.deepEqual(stopped, { code: 0, signal: null });
  assert.equal(contents('forwarded'), 'forwarded');
  assert.ok(output.includes('Stopped'));
  console.log('PASS: POSIX supervisor forwards termination and waits for graceful child shutdown');

  const batch = text('start.bat');
  assert.ok(batch.includes('chcp 65001'));
  for (const name of ['missing', 'normal', 'restart', 'failed']) {
    const block = batch.slice(batch.indexOf(`if "%~1"=="${name}"`)).split('\n)')[0];
    for (const language of languages.filter(l => l.id !== 'en')) assert.ok(block.includes(`"${language.id}"`), `${name}: Windows ${language.id}`);
  }
  for (const line of ['"%BIN%" %*', 'set "code=!ERRORLEVEL!"', 'set "RESTART_CODE=75"', 'set "MAX_DELAY=60"', 'ping -n !pings! 127.0.0.1 >nul 2>&1', 'set /a delay=!delay!*2']) assert.ok(batch.includes(line));
  console.log('PASS: Windows UTF-8, six-language message branches and supervisor control-flow inspection (native execution unverified)');
} finally {
  rmSync(fixture, { recursive: true, force: true });
}
