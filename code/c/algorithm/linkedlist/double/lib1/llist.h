#ifndef LLIST_H__
#define LLIST_H__

#define LLIST_FORWARD 1
#define LLIST_BACKWARD 2

typedef void llist_op(const void *);
typedef int llist_cmp(const void *, const void *);

struct llist_node_st
{
    void *data;
    struct llist_node_st *prev;
    struct llist_node_st *next;
};

typedef struct
{
    int size;
    struct llist_node_st head;
} LLIST;

LLIST *llist_create(int); //带头节点的空的双向链表
void llist_travel(LLIST *, llist_op *);
void llist_destroy(LLIST *);
int llist_insert(LLIST *, const void *, int); //int mode
void *llist_find(LLIST *, const void *, llist_cmp *);
int llist_delete(LLIST *, const void *, llist_cmp *);
int llist_fetch(LLIST *, const void *, llist_cmp *, void *);

#endif