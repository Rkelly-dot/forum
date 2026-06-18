// ── Like / Dislike ────────────────────────────────────────
document.addEventListener('click', async (e) => {
  const btn = e.target.closest('[data-like]');
  if (!btn) return;
  e.preventDefault();

  const postId    = btn.dataset.postId    || '';
  const commentId = btn.dataset.commentId || '';
  const value     = btn.dataset.like;

  const res = await fetch('/like', {
    method: 'POST',
    headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
    body: new URLSearchParams({ post_id: postId, comment_id: commentId, value }),
  });

  if (res.ok) {
    const { likes, dislikes } = await res.json();
    const card = btn.closest('.post-card, .comment-card');
    if (card) {
      card.querySelector('.vote-count').textContent = likes - dislikes;
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