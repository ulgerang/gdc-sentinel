# AGENTS.md — AI 코딩 에이전트 가이드

이 파일은 AI 코딩 에이전트(Claude Code, Cursor, Copilot 등)가 이 프로젝트에서 작업할 때 참고하는 컨텍스트입니다.

## 프로젝트 개요

GDC Sentinel은 GDC(Graph-Driven Codebase) CLI의 컴패니언 도구입니다.
GDC의 JSON 출력을 소비하여 드리프트를 탐지하고, 컨텍스트 패킷을 생성하며, 외부 코딩 에이전트를 오케스트레이션합니다.

## 아키텍처

```
main.go → cli.Execute() (cobra)
  ├── init   → config.DefaultConfig() → 파일 시스템 생성
  ├── scan   → git diff -M → gdc.Client.Query/Diff → inbox 저장
  ├── packet → gdc.Client.Context/Refs/Deps + inbox + note → markdown
  ├── run    → packet 생성 → exec.Command(에이전트) → RunLog 저장
  ├── note   → note 패키지 (파일 기반 CRUD)
  └── explain → gdc.Client 쿼리 + inbox + note → 종합 출력
```

## 핵심 제약

1. **ID 생성**: `internal/idgen` 패키지 사용. `crypto/rand` 기반 16-hex. 절대 `time.Now().UnixNano()` 사용 금지 (충돌 위험).
2. **에이전트 명령어 검증**: `run.ValidateCommand`가 셸 메타문자(`;&|`$()<>\n\r`)를 차단함. `exec.Command` 사용 — 셸 인젝션 불가.
3. **설정 주입**: `gdc.Client`는 하드코딩된 명령어 없이 `cfg.GDCCommands()`에서 주입받음.
4. **에러 처리**: silent `_ = err` 금지. 반드시 `printWarning` 또는 반환.
5. **InputMode**: `file`(args에 `{{packet_path}}`), `stdin`(파이프), `argument`(마지막 arg), 미지원 → 에러.

## 파일 위치

| 데이터 | 경로 |
|--------|------|
| 설정 | `.gdc-sentinel/config.yaml` |
| 드리프트 아이템 | `.gdc-sentinel/inbox/*.json` |
| 노트 | `.gdc-sentinel/notes/{node}/*.json` |
| 패킷 | `.gdc-sentinel/packets/*.md` |
| 실행 로그 | `.gdc-sentinel/run_logs/*.json` |

## 빌드/테스트 명령

```bash
make build          # go build -o gdc-sentinel .
make test           # go test ./...
go vet ./...        # 정적 분석
```

## GDC CLI 의존성

`gdc` 바이너리가 PATH에 있어야 `scan`, `packet`, `run`, `explain`이 동작합니다.
GDC CLI 소스: `/home/ulgerang/works/gdc`

## 의존성

- `github.com/spf13/cobra` — CLI 프레임워크
- `github.com/fatih/color` — 컬러 출력
- `gopkg.in/yaml.v3` — YAML 파싱

외부 의존성 추가는 최소화합니다.
