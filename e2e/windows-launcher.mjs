#!/usr/bin/env node
// Native cmd.exe proof using only a copied launcher and local fixture executables.
import assert from 'node:assert/strict';
import { copyFileSync, existsSync, mkdtempSync, readFileSync, rmSync, writeFileSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { dirname, join, resolve } from 'node:path';
import { spawnSync } from 'node:child_process';
import { fileURLToPath } from 'node:url';

assert.equal(process.platform, 'win32', 'Native Windows is required; use node --check on other platforms.');
const repo = resolve(dirname(fileURLToPath(import.meta.url)), '..');
const fixture = mkdtempSync(join(tmpdir(), 'scopeweaver-windows launcher-'));
const messages = {
  en: ['Executable not found: scopeweaver.exe', 'Exited normally', 'Restart requested; applying update...', 'Exited with code 7; restarting in 1s'],
  ko: ['실행 파일을 찾을 수 없습니다: scopeweaver.exe', '정상 종료', '다시 시작 요청; 업데이트 적용 중...', '종료 코드 7; 1초 후 다시 시작'],
  'zh-Hans': ['未找到可执行文件：scopeweaver.exe', '已正常退出', '收到重启请求；正在应用更新...', '退出码为 7；1 秒后重启'],
  'zh-Hant': ['找不到執行檔：scopeweaver.exe', '已正常結束', '收到重新啟動要求；正在套用更新...', '結束代碼為 7；1 秒後重新啟動'],
  ja: ['実行ファイルが見つかりません: scopeweaver.exe', '正常終了しました', '再起動が要求されました。更新を適用中...', '終了コード 7。1 秒後に再起動します'],
  es: ['No se ha encontrado el ejecutable: scopeweaver.exe', 'Finalizó normalmente', 'Se ha solicitado reiniciar; aplicando la actualización...', 'Finalizó con el código 7; se reiniciará en 1 s'],
};
const definitions = JSON.parse(readFileSync(join(repo, 'locale/definitions.json')));
assert.deepEqual(Object.keys(messages), definitions.locales.map(language => language.id));
const vectors = JSON.parse(readFileSync(join(repo, 'locale/matching-vectors.json'))).vectors;
const pathKey = Object.keys(process.env).find(key => key.toLowerCase() === 'path') || 'Path';
const env = { ...process.env, [pathKey]: `${fixture};${process.env[pathKey] || ''}`, ARTEX_LANGUAGE: '', FIXTURE_ROOT: fixture };
const launch = (language, codes = [0]) => spawnSync(process.env.ComSpec || 'cmd.exe', [
  '/d', '/s', '/c', 'start.bat "argument with spaces" "--literal=*"',
], {
  cwd: fixture, env: { ...env, SCOPEWEAVER_LANGUAGE: language, FIXTURE_CODES: JSON.stringify(codes) },
  encoding: 'utf8', timeout: 30000, windowsHide: true, windowsVerbatimArguments: true,
});
const success = result => assert.equal(result.status, 0, result.error?.message || result.stderr || result.stdout);
const contents = name => readFileSync(join(fixture, name), 'utf8');
const clear = () => { for (const name of ['count', 'args.jsonl', 'pings.jsonl']) rmSync(join(fixture, name), { force: true }); };
try {
  copyFileSync(join(repo, 'start.bat'), join(fixture, 'start.bat'));
  // The same tiny Go executable serves as the child and as ping.exe. Neither
  // opens a network connection, installs anything or runs application code.
  writeFileSync(join(fixture, 'fixture.go'), `package main
import("encoding/json";"os";"path/filepath";"strconv";"strings")
func must(err error){if err!=nil{panic(err)}}
func record(name string){data,err:=json.Marshal(os.Args[1:]);must(err);file,err:=os.OpenFile(filepath.Join(os.Getenv("FIXTURE_ROOT"),name),os.O_CREATE|os.O_WRONLY|os.O_APPEND,0600);must(err);_,err=file.Write(append(data,'\\n'));must(err);must(file.Close())}
func main(){root:=os.Getenv("FIXTURE_ROOT");executable,err:=os.Executable();must(err);if strings.EqualFold(filepath.Base(executable),"ping.exe"){record("pings.jsonl");return};n:=0;if data,err:=os.ReadFile(filepath.Join(root,"count"));err==nil{n,err=strconv.Atoi(string(data));must(err)}else if !os.IsNotExist(err){must(err)};record("args.jsonl");must(os.WriteFile(filepath.Join(root,"count"),[]byte(strconv.Itoa(n+1)),0600));var codes []int;must(json.Unmarshal([]byte(os.Getenv("FIXTURE_CODES")),&codes));if n<len(codes){os.Exit(codes[n])}}
`);
  const built = spawnSync('go', ['build', '-o', 'fixture.exe', 'fixture.go'], {
    cwd: fixture, env: { ...env, GOWORK: 'off', GO111MODULE: 'off', GOOS: 'windows', CGO_ENABLED: '0' },
    encoding: 'utf8', timeout: 120000, windowsHide: true,
  });
  success(built);
  assert.ok(existsSync(join(fixture, 'fixture.exe')));
  // cmd.exe resolves this local executable before the system ping. Identify it
  // by os.Executable(): its command-line argv[0] can remain extensionless "ping".
  copyFileSync(join(fixture, 'fixture.exe'), join(fixture, 'ping.exe'));
  const ping = spawnSync(process.env.ComSpec || 'cmd.exe', ['/d', '/s', '/c', 'ping -n 2 127.0.0.1'], {
    cwd: fixture, env, encoding: 'utf8', timeout: 30000, windowsHide: true, windowsVerbatimArguments: true,
  });
  success(ping);
  assert.deepEqual(JSON.parse(contents('pings.jsonl')), ['-n', '2', '127.0.0.1']);
  assert.ok(!existsSync(join(fixture, 'count')), 'extensionless ping was mistaken for the child');
  clear();
  for (const [language, [missing, normal, restart, failed]] of Object.entries(messages)) {
    rmSync(join(fixture, 'scopeweaver.exe'), { force: true });
    clear();
    const absent = launch(language);
    assert.equal(absent.status, 1, absent.error?.message || absent.stderr || absent.stdout);
    assert.equal(absent.stderr.trim(), `[ScopeWeaver] ${missing}`);
    assert.equal(absent.stdout.trim(), '');
    assert.ok(!existsSync(join(fixture, 'count')), 'missing child was invoked');
    copyFileSync(join(fixture, 'fixture.exe'), join(fixture, 'scopeweaver.exe'));
    const exited = launch(language);
    success(exited);
    assert.equal(exited.stdout.trim(), `[ScopeWeaver] ${normal}`);
    assert.equal(exited.stderr.trim(), '');
    assert.equal(contents('count'), '1');
    assert.deepEqual(JSON.parse(contents('args.jsonl')), ['argument with spaces', '--literal=*']);
    clear();
    const restarted = launch(language, [75, 7, 7, 7, 7, 7, 7, 7, 7, 75, 7, 0]);
    success(restarted);
    assert.equal(contents('count'), '12');
    assert.equal(restarted.stdout.split(`[ScopeWeaver] ${restart}`).length - 1, 2);
    assert.ok(restarted.stdout.trim().endsWith(`[ScopeWeaver] ${normal}`));
    assert.ok(restarted.stderr.includes(`[ScopeWeaver] ${failed}`));
    for (const line of contents('args.jsonl').trim().split(/\r?\n/)) assert.deepEqual(JSON.parse(line), ['argument with spaces', '--literal=*']);
    const pings = contents('pings.jsonl').trim().split(/\r?\n/).map(line => JSON.parse(line));
    assert.deepEqual(pings, [2, 3, 5, 9, 17, 33, 61, 61, 2].map(count => ['-n', String(count), '127.0.0.1']));
    console.log(`PASS: ${language} native missing/normal/restart/failure messages, arguments, capped backoff and exit-75 reset`);
  }
  rmSync(join(fixture, 'scopeweaver.exe'));
  for (const { input, expected } of vectors) {
    const absent = launch(input);
    assert.equal(absent.status, 1, absent.error?.message || absent.stderr || absent.stdout);
    assert.equal(absent.stderr.trim(), `[ScopeWeaver] ${messages[expected || 'en'][0]}`, input);
  }
  console.log(`PASS: ${vectors.length} shared matching vectors through native Windows missing-binary output`);
} finally {
  rmSync(fixture, { recursive: true, force: true });
}
