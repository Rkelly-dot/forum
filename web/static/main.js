// ── Like / Dislike ────────────────────────────────────────
document.addEventListener('click', async (e) => {
  const btn = e.target.closest('[data-like]');
  if (!btn) return;
  e.preventDefault();

  const postId    = btn.dataset.postId    || '';
  const commentId = btn.dataset.commentId || '';
  const value     = btn.dataset.like;

  const res = await fetch('/posts/like', {
    method: 'POST',
    headers: { 'Content-Type': 'application/x-www-form-urlencoded', 'Accept': 'application/json' },
    body: new URLSearchParams({ post_id: postId, comment_id: commentId, value }),
  });

  if (res.ok) {
    const { likes, dislikes } = await res.json();
    const card = btn.closest('.post-card, .comment-card');
    if (card) {
      const likeEl = card.querySelector('.like-count');
      const dislikeEl = card.querySelector('.dislike-count');
      if (likeEl) likeEl.textContent = likes;
      if (dislikeEl) dislikeEl.textContent = dislikes;
      // fallback for older templates
      const voteEl = card.querySelector('.vote-count');
      if (voteEl && !likeEl && !dislikeEl) voteEl.textContent = likes - dislikes;
    }
  }
});

// ── Filter buttons ────────────────────────────────────────
document.querySelectorAll('.filter-btn').forEach(btn => {
  btn.addEventListener('click', () => {
    document.querySelectorAll('.filter-btn').forEach(b => b.classList.remove('active'));
    btn.classList.add('active');
  });
});