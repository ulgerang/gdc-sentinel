# GDC Sentinel

GDC(Graph-Driven Codebase)를 위한 드리프트 탐지 및 에이전트 오케스트레이션 도구.

## 기능

| 명령어 | 설명 |
|--------|------|
| `init` | `.gdc-sentinel/` 설정 트리 생성 |
| `scan` | `git diff`로 변경 파일 탐지 → GDC 쿼리 → 드리프트 아이템 저장 |
| `packet` | 노드의 컨텍스트 + 드리프트 + 노트를 마크다운 패킷으로 렌더 |
| `run` | 외부 코딩 에이전트(CLI)에 패킷 전달 후 실행 |
| `note` | 노드별 로컬 노트 관리 (추가/목록/삭제) |
| `explain` | 노드의 의존성, 참조, 드리프트, 노트를 종합 출력 |

## 설치

```bash
git clone https://github.com/ulgerang/gdc-sentinel.git
cd gdc-sentinel
make build
```

Go 1.23+ 필요.

## 빠른 시작

```bash
# 1. 프로젝트 초기화
gdc-sentinel init

# 2. 드리프트 스캔
gdc-sentinel scan

# 3. 특정 노드의 컨텍스트 패킷 생성
gdc-sentinel packet --node my.service.UserRepo

# 4. 에이전트 실행
gdc-sentinel run --node my.service.UserRepo --agent default

# 5. 노드 설명 보기
gdc-sentinel explain --node my.service.UserRepo
```

## 설정

`gdc-sentinel init`이 `.gdc-sentinel/config.yaml`을 생성합니다.

```yaml
version: 1
project:
  name: my-project
  path: .
  gdc_command: gdc
  default_since: 24h
gdc:
  commands:
    query:   ["query", "{symbol}", "--format", "json"]
    deps:    ["deps", "{node}", "--depth", "{depth}"]
    refs:    ["refs", "{node}", "--depth", "{depth}"]
    context: ["context", "{node}", "--with-impl", "--with-tests", "--with-callers"]
    diff:    ["diff", "{node}", "--format", "json"]
agents:
  default:
    type: cli
    command: echo
    args: ["{{packet_path}}"]
    input_mode: file
    working_dir: .
    timeout_seconds: 300
scan:
  include: ["**/*.go", "**/*.ts", "**/*.tsx"]
  exclude: ["vendor/**", "node_modules/**", ".git/**"]
drift_policy:
  auto_write_specs: false
  require_human_approval: true
  confidence_threshold: high
```

### 에이전트 설정

`input_mode`에 따라 패킷 전달 방식이 다릅니다:

| 모드 | 동작 |
|------|------|
| `file` | `{{packet_path}}`를 args에 직접 포함해 파일 경로로 전달 |
| `stdin` | 패킷 내용을 stdin으로 파이프 |
| `argument` | 패킷 내용을 마지막 arg로 추가 |

## 명령어 상세

### `init [--force]`

`.gdc-sentinel/` 디렉토리와 기본 설정 파일을 생성합니다.

- `--force`: 기존 `.gdc-sentinel/` 내용을 모두 삭제 후 재생성

### `scan [--verbose] [--since DURATION]`

Git 변경 사항을 탐지하고 GDC에 쿼리하여 드리프트를 기록합니다.

- `--since`: 스캔 기간 (기본값: 설정의 `default_since`)
- `--verbose`: 상세 로그 출력
- rename 탐지를 위해 `git diff -M --name-status` 사용

### `packet --node SYMBOL`

지정한 노드의 컨텍스트 패킷을 마크다운으로 생성합니다.

생성 위치: `.gdc-sentinel/packets/{timestamp}_{node}.md`

### `run --node SYMBOL [--agent NAME]`

패킷을 생성하고 설정된 에이전트에 전달합니다.

- `--agent`: 사용할 에이전트 (기본값: `default`)
- 명령어 유효성 검사: 셸 메타문자 차단, 경로 이스케이프 방지

### `note --node SYMBOL [--add TEXT] [--delete ID] [--list]`

노드별 노트를 관리합니다.

### `explain --node SYMBOL`

노드의 종합 정보를 출력합니다 (의존성, 참조, 드리프트 아이템, 노트).

## 프로젝트 구조

```
.
├── main.go
├── internal/
│   ├── cli/          # CLI 명령어 (cobra)
│   ├── config/       # YAML 설정 로더
│   ├── gdc/          # GDC CLI 래퍼
│   ├── idgen/        # crypto/rand 기반 ID 생성
│   ├── inbox/        # 드리프트 아이템 저장소
│   ├── note/         # 노트 저장소
│   ├── packet/       # 마크다운 패킷 렌더러
│   └── run/          # 에이전트 실행기
└── .gdc-sentinel/    # 런타임 데이터 (git 추적 제외)
    ├── config.yaml
    ├── inbox/
    ├── notes/
    └── packets/
```

## 개발

```bash
make build          # 빌드
make test           # 테스트
make clean          # 정리
```

## 라이선스

MIT
