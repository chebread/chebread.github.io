---
date: 2024-07-23 
category: [전체, 에세이] 
published: true
thumbnail: https://cdn.chebread.org/blog/adam-wathan.webp
---

<img src="https://cdn.chebread.org/blog/tailwind-css-1.webp" width="300" />

위와 같은 반응형 레이아웃을 만들기 위해서 Tailwind CSS는 이 정도의 코드를 요구한다.

```jsx
<main class="py-6 px-4 sm:p-6 md:py-10 md:px-8">
  <div class="max-w-4xl mx-auto grid grid-cols-1 lg:max-w-5xl lg:gap-x-20 lg:grid-cols-2">
    <div class="relative p-3 col-start-1 row-start-1 flex flex-col-reverse rounded-lg bg-gradient-to-t from-black/75 via-black/0 sm:bg-none sm:row-start-2 sm:p-0 lg:row-start-1">
      <h1 class="mt-1 text-lg font-semibold text-white sm:text-slate-900 md:text-2xl dark:sm:text-white">Beach House in Collingwood</h1>
      <p class="text-sm leading-4 font-medium text-white sm:text-slate-500 dark:sm:text-slate-400">Entire house</p>
    </div>
    <div class="grid gap-4 col-start-1 col-end-3 row-start-1 sm:mb-6 sm:grid-cols-4 lg:gap-6 lg:col-start-2 lg:row-end-6 lg:row-span-6 lg:mb-0">
      <img src="/beach-house.jpg" alt="" class="w-full h-60 object-cover rounded-lg sm:h-52 sm:col-span-2 lg:col-span-full" loading="lazy">
      <img src="/beach-house-interior-1.jpg" alt="" class="hidden w-full h-52 object-cover rounded-lg sm:block sm:col-span-2 md:col-span-1 lg:row-start-2 lg:col-span-2 lg:h-32" loading="lazy">
      <img src="/beach-house-interior-2.jpg" alt="" class="hidden w-full h-52 object-cover rounded-lg md:block lg:row-start-2 lg:col-span-2 lg:h-32" loading="lazy">
    </div>
    <dl class="mt-4 text-xs font-medium flex items-center row-start-2 sm:mt-1 sm:row-start-3 md:mt-2.5 lg:row-start-2">
      <dt class="sr-only">Reviews</dt>
      <dd class="text-indigo-600 flex items-center dark:text-indigo-400">
        <svg width="24" height="24" fill="none" aria-hidden="true" class="mr-1 stroke-current dark:stroke-indigo-500">
          <path d="m12 5 2 5h5l-4 4 2.103 5L12 16l-5.103 3L9 14l-4-4h5l2-5Z"  stroke-width="2" stroke-linecap="round" stroke-linejoin="round" />
        </svg>
        <span>4.89 <span class="text-slate-400 font-normal">(128)</span></span>
      </dd>
      <dt class="sr-only">Location</dt>
      <dd class="flex items-center">
        <svg width="2" height="2" aria-hidden="true" fill="currentColor" class="mx-3 text-slate-300">
          <circle cx="1" cy="1" r="1" />
        </svg>
        <svg width="24" height="24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="mr-1 text-slate-400 dark:text-slate-500" aria-hidden="true">
          <path d="M18 11.034C18 14.897 12 19 12 19s-6-4.103-6-7.966C6 7.655 8.819 5 12 5s6 2.655 6 6.034Z" />
          <path d="M14 11a2 2 0 1 1-4 0 2 2 0 0 1 4 0Z" />
        </svg>
        Collingwood, Ontario
      </dd>
    </dl>
    <div class="mt-4 col-start-1 row-start-3 self-center sm:mt-0 sm:col-start-2 sm:row-start-2 sm:row-span-2 lg:mt-6 lg:col-start-1 lg:row-start-3 lg:row-end-4">
      <button type="button" class="bg-indigo-600 text-white text-sm leading-6 font-medium py-2 px-3 rounded-lg">Check availability</button>
    </div>
    <p class="mt-4 text-sm leading-6 col-start-1 sm:col-span-2 lg:mt-6 lg:row-start-4 lg:col-span-1 dark:text-slate-400">
      This sunny and spacious room is for those traveling light and looking for a comfy and cosy place to lay their head for a night or two. This beach house sits in a vibrant neighborhood littered with cafes, pubs, restaurants and supermarkets and is close to all the major attractions such as Edinburgh Castle and Arthur's Seat.
    </p>
  </div>
</main>
```

나는 여러분들에게 물어보고자 한다. 이것이 과연 정상적인 코드인가?
도대체 이 코드를 보고 레이아웃이 머릿속으로 그려지기는 하는가?

내가 Tailwind CSS className에 익숙하지 않아서 그럴 수도 있을 것이다.
그러나 당신이 과연 기존의 CSS 스타일링으로 넘어갈 수 있는가?
className이 온통 약자로 되어있어서 결국 기존의 속성값이 무엇인지 까먹게 되지 않는가?
나는 당신이 기존의 CSS 스타일링으로 넘어가는 데에 많은 시간을 소요할 것이라고 단연코 생각한다.

그리고 유의할 점은 Tailwind CSS는 CSS를 인라인상으로 표현하게 해주는 도구가 아니다.
Tailwind CSS는 CSS 용 부트스트랩이다.
다른 말로, Tailwind CSS는 제한된 CSS 스타일 셋이다.

당신이 Tailwind CSS에 지나치게 의존한다면, 깊은 의미로는 그만큼의 CSS 스타일링 실력이 없다는 것이다.

<img src="https://cdn.chebread.org/blog/tailwind-css-2.webp" width="300" />

위처럼 이미 모든 스타일은 Tailwind CSS 상에서 정해져 있다. 그리고 개발자는 이런 정의된 스타일을 차용하여 프론트엔드를 구성한다. 그리고 정의되지 않은 스타일은 따로 관리한다.

도대체 이것이 복제가 아니면 무엇인가?
프론트엔드 개발자는 UI/UX를 개발하는 사람 아닌가?
왜 이미 정의된 UI/UX의 경험을 복사/붙여넣기 하는 것인가?

더불어 Tailwind CSS의 최대 단점은 기존의 CSS 스타일링 방식을 모두 파괴했다는 것이다.
당신이 처음 Tailwind CSS를 접하는 프론트엔드 개발자라면 도대체 이 코드를 해석할 수 있는가?

그리고 재사용성을 지원하지 않는다는 것이다.
Tailwind CSS는 재사용을 하기 위해서는 멀티 커서를 사용하라고 한다.

![](https://cdn.chebread.org/blog/tailwind-css-3.webp)

아니면 루프를 사용하라고 한다.

![](https://cdn.chebread.org/blog/tailwind-css-4.webp)

아니면 @apply라는 기능을 활용하라고 한다.

```css
@tailwind base;
@tailwind components;
@tailwind utilities;

@layer components {
  .btn-primary {
    @apply py-2 px-5 bg-violet-500 text-white font-semibold rounded-full shadow-md hover:bg-violet-700 focus:outline-none focus:ring focus:ring-violet-400 focus:ring-opacity-75;
  }
}
```

또한, Tailwind CSS로 구현하기 복잡한 스타일링이 너무 많다는 점이다.
당신이 복잡한 스타일링을 구현하기 위해서는 배보다 배꼽이 더 큰 상황을 목도할 수 있다.

결론적으로 당신이 Tailwind CSS에만 익숙해지게 되면 당신은 더욱더 창의적인 레이아웃을 구현할 수 없을 것이다. Tailwind CSS는 아주 빠른 기일 내에 프로젝트를 구현하는 데에만 아주 적합하므로, 유지 보수의 관점에서는 정말 끔찍하다.

나는 이것이 과연 프론트엔드가 지향하는 관점이 맞는가 묻고 싶다.

---

> 섬네일 출처: https://tailwindcss.com/blog/2023-07-18-tailwind-connect-2023-recap
