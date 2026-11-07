import mediumZoom from 'medium-zoom';

function initializeStickyNav() {
  const navs = document.querySelectorAll('.home-nav, .about-nav, .post-nav-hero');

  if (navs.length === 0) return;

  window.addEventListener('scroll', () => {
    navs.forEach(nav => {
      if (window.scrollY > 10) {
        nav.classList.add('scrolled');
      } else {
        nav.classList.remove('scrolled');
      }
    });
  });
}

function initializeBackLink() {
  const backLinks = document.querySelectorAll('#back');

  backLinks.forEach(link => {
    link.addEventListener('click', (event) => {
      event.preventDefault();
      history.back();
    });
  });
}

function initializeHeroEdgeColors() {
  const hero = document.querySelector('.home-hero') as HTMLElement | null;
  if (!hero) return;

  const heroStyle = window.getComputedStyle(hero);
  const heroAfterStyle = window.getComputedStyle(hero, '::after');
  const bgImage = (heroStyle.backgroundImage && heroStyle.backgroundImage !== 'none')
    ? heroStyle.backgroundImage
    : (heroAfterStyle.backgroundImage || '');
  const match = bgImage.match(/url\(["']?([^"')]+)["']?\)/);
  if (!match) return;

  const img = new Image();
  img.crossOrigin = 'anonymous';
  img.src = match[1];

  img.onload = () => {
    try {
      const canvas = document.createElement('canvas');
      canvas.width = img.naturalWidth;
      canvas.height = img.naturalHeight;
      const ctx = canvas.getContext('2d');
      if (!ctx) return;

      ctx.drawImage(img, 0, 0);

      const h = canvas.height;
      const w = canvas.width;
      const sampleW = Math.max(1, Math.min(20, Math.floor(w * 0.05)));

      // 사진 가장자리의 상하 수직 색상 변화를 4단계(상단, 상중, 하중, 하단)로 샘플링하여
      // 좌우 윙 배경이 사진의 실제 수직 광원/색조와 1:1로 일치하도록 자연스러운 수직 그라데이션 생성
      const sampleSlice = (x: number, yStart: number, yEnd: number) => {
        const sliceH = Math.max(1, yEnd - yStart);
        const data = ctx.getImageData(x, yStart, sampleW, sliceH).data;
        let r = 0, g = 0, b = 0;
        const count = data.length / 4;
        for (let i = 0; i < data.length; i += 4) {
          r += data[i];
          g += data[i + 1];
          b += data[i + 2];
        }
        return `rgb(${Math.round(r / count)}, ${Math.round(g / count)}, ${Math.round(b / count)})`;
      };

      const y1 = Math.floor(h * 0.25);
      const y2 = Math.floor(h * 0.55);
      const y3 = Math.floor(h * 0.85);

      const lTop = sampleSlice(0, 0, y1);
      const lMidTop = sampleSlice(0, y1, y2);
      const lMidBot = sampleSlice(0, y2, y3);
      const lBot = sampleSlice(0, y3, h);

      const rTop = sampleSlice(w - sampleW, 0, y1);
      const rMidTop = sampleSlice(w - sampleW, y1, y2);
      const rMidBot = sampleSlice(w - sampleW, y2, y3);
      const rBot = sampleSlice(w - sampleW, y3, h);

      const leftGrad = `linear-gradient(to bottom, ${lTop} 0%, ${lMidTop} 35%, ${lMidBot} 70%, ${lBot} 100%)`;
      const rightGrad = `linear-gradient(to bottom, ${rTop} 0%, ${rMidTop} 35%, ${rMidBot} 70%, ${rBot} 100%)`;

      document.documentElement.style.setProperty('--hero-edge-left-gradient', leftGrad);
      document.documentElement.style.setProperty('--hero-edge-right-gradient', rightGrad);
    } catch {
      // CORS 또는 에러 발생 시 CSS 기본값 유지
    }
  };
}

document.addEventListener('DOMContentLoaded', () => {
    initializeStickyNav();
    initializeBackLink();
    initializeHeroEdgeColors();
    mediumZoom('.markdown-body img', {
         background: 'var(--bg-page)',
       });
});
