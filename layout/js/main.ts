import mediumZoom from 'medium-zoom';

function initializeStickyNav() {
  const navs = document.querySelectorAll('.home-nav, .about-nav, .post-nav-hero');

  if (navs.length === 0) return;

  const updateScrolled = () => {
    const isScrolled = window.scrollY > 10;
    navs.forEach(nav => {
      if (isScrolled) {
        nav.classList.add('scrolled');
      } else {
        nav.classList.remove('scrolled');
      }
    });
  };

  window.addEventListener('scroll', updateScrolled, { passive: true });
  updateScrolled();
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

function getTargetElementByHash(hash: string): HTMLElement | null {
  if (!hash.startsWith('#') || hash.length < 2) return null;

  const rawId = hash.slice(1);
  let decodedId: string;
  try {
    decodedId = decodeURIComponent(rawId);
  } catch {
    decodedId = rawId;
  }

  return document.getElementById(decodedId) || document.getElementById(rawId);
}

function scrollElementToViewportPosition(target: HTMLElement): void {
  const nav = document.querySelector<HTMLElement>('.post-nav, .home-nav, .about-nav');
  const navHeight = nav ? nav.getBoundingClientRect().height : 0;

  const rect = target.getBoundingClientRect();
  const currentScrollY = typeof window.scrollY === 'number' ? window.scrollY : window.pageYOffset;
  const targetViewportY = Math.max(navHeight + 16, window.innerHeight * 0.45);
  const targetCenter = rect.top + currentScrollY + rect.height / 2;
  const maxScrollY = Math.max(0, document.documentElement.scrollHeight - window.innerHeight);
  const targetTop = Math.min(maxScrollY, Math.max(0, targetCenter - targetViewportY));

  window.scrollTo(0, targetTop);
}

function initializeAnchorScroll(): void {
  const anchorLinks = document.querySelectorAll<HTMLAnchorElement>('.markdown-body a[href^="#"]');

  anchorLinks.forEach(link => {
    link.addEventListener('click', (event) => {
      if (
        event.defaultPrevented ||
        event.button !== 0 ||
        event.metaKey ||
        event.ctrlKey ||
        event.shiftKey ||
        event.altKey
      ) {
        return;
      }

      const href = link.getAttribute('href');
      if (!href || href === '#') return;

      const target = getTargetElementByHash(href);
      if (!target) return;

      event.preventDefault();
      if (window.location.hash !== href) {
        history.pushState(null, '', href);
      }
      scrollElementToViewportPosition(target);
    });
  });
}

document.addEventListener('DOMContentLoaded', () => {
    initializeStickyNav();
    initializeBackLink();
    initializeAnchorScroll();
    mediumZoom('.markdown-body img', {
         background: 'var(--bg-page)',
       });
});
