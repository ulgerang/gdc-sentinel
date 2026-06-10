# 기여 가이드

## 개발 환경

- Go 1.23+
- `make build` / `make test`

## 코드 구조

```
internal/
├── cli/       # 모든 CLI 명령어. cobra 기반.
├── config/    # YAML 설정 읽기/쓰기.
├── gdc/       # GDC CLI JSON 출력 래퍼.
├── idgen/     # crypto/rand 기반 16-hex ID 생성.
├── inbox/     # 드리프트 아이템 파일 저장소.
├── note/      # 노트 파일 저장소.
├── packet/    # 마크다운 패킷 렌더링.
└── run/       # 외부 에이전트 실행 및 검증.
```

## 커밋 규칙

```
타입: 간단한 설명

<선택적 상세 내용>
```

타입: `feat`, `fix`, `refactor`, `test`, `docs`, `chore`

## PR 프로세스

1. 기능 브랜치 생성: `feat/기능명` 또는 `fix/버그명`
2. `make test` 통과 확인
3. `go vet ./...` 워닝 없음 확인
4. PR 생성

## 테스트

모든 패키지에 테스트가 있습니다. 새 기능은 반드시 테스트를 동반해야 합니다.

```bash
make test
```

## 코딩 컨벤션

- `gofmt` / `goimports` 준수
- 외부 패키지 의존성 최소화
- 에러는 항상 명시적으로 처리 (silent `_ =` 금지)
- 공개 함수/타입은 godoc으로 문서화
