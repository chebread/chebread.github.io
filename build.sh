#!/bin/bash

set -e

run_step() {
  local step_name="$1"
  shift

  echo "running: ${step_name}..."

  if "$@"; then
    echo "success: ${step_name} 성공"
  else
    echo "warning: ${step_name} 실패" >&2
    exit 1
  fi
}

echo "-- 빌드 시작 --"

mkdir -p public
run_step "public 디렉토리 초기화" find public/ -mindepth 1 -delete

run_step "sass 컴파일" pnpm sass layout/styles:public/styles --style=compressed

run_step "esbuild 컴파일" pnpm esbuild layout/js/main.ts --bundle --outfile=public/js/main.js --minify --target=es2017

run_step ".nojekyll 파일 생성" touch public/.nojekyll

run_step "robots.txt 파일 복사" cp layout/robots.txt public/robots.txt

run_step "assets 디렉토리 복사" cp -r assets/ public/assets

run_step "favicons 디렉토리 복사" cp -r layout/favicons public/favicons

if [ "$1" = "production" ]; then
  run_step "프로덕션 모드로 go 실행" pnpm cross-env APP_ENV=production go run .
else
  run_step "개발 모드로 go 실행" go run .
fi

echo "-- 빌드 완료 --"
