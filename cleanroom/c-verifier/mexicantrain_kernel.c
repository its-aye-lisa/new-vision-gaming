/*
 * Clean-Room Independent Reference Kernel for Mexican Train Dominoes
 * Compiled independently under BOTH gcc and clang (-Wall -Wextra -Werror)
 * to verify 3-way behavioral equivalence against the pure Go kernel.
 *
 * Usage:
 *   mexicantrain_kernel <turn> <openDoubleTrain> <ownOpenEnd> <mexOpenEnd> <oppOpenEnd> \
 *     <oppPublic:0|1> <action:PLAY|PASS|DRAW> <targetTrain> <tileLeft> <tileRight> \
 *     <handCount> <boneyardCount> <drawnThisTurn:0|1> <actor>
 *
 * Output:
 *   <VALID|INVALID> <newOpenEnd> <nextTurn> <newOpenDouble> <ownPublic:0|1> <handOver:0|1> <reason>
 */
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

int main(int argc, char **argv) {
    if (argc < 15) {
        fprintf(stderr, "usage: mexicantrain_kernel <14 args>\n");
        return 2;
    }

    int turn = atoi(argv[1]);
    int open_double = atoi(argv[2]);
    int own_open = atoi(argv[3]);
    int mex_open = atoi(argv[4]);
    int opp_open = atoi(argv[5]);
    int opp_public = atoi(argv[6]) != 0 ? 1 : 0;
    const char *action = argv[7];
    int target = atoi(argv[8]);
    int tile_l = atoi(argv[9]);
    int tile_r = atoi(argv[10]);
    int hand_count = atoi(argv[11]);
    int boneyard = atoi(argv[12]);
    int drawn = atoi(argv[13]) != 0 ? 1 : 0;
    /* argv[14] is actor — intentionally IGNORED per MT-INV-6 (anti-backdoor determinism) */
    (void)argv[14];

    if (turn != 0 && turn != 1) {
        printf("INVALID %d %d %d 0 0 ERR_INVALID_TURN\n", own_open, turn, open_double);
        return 0;
    }
    int opp = 1 - turn;

    if (strcmp(action, "PASS") == 0) {
        if (boneyard > 0 && !drawn) {
            printf("INVALID %d %d %d 0 0 ERR_MUST_DRAW_FIRST\n", own_open, turn, open_double);
            return 0;
        }
        printf("VALID %d %d %d 1 0 TURN_PASSED_TRAIN_OPENED\n", own_open, opp, open_double);
        return 0;
    }

    if (strcmp(action, "DRAW") == 0) {
        if (drawn) {
            printf("INVALID %d %d %d 0 0 ERR_ALREADY_DRAWN\n", own_open, turn, open_double);
            return 0;
        }
        if (boneyard <= 0) {
            printf("INVALID %d %d %d 0 0 ERR_BONEYARD_EMPTY\n", own_open, turn, open_double);
            return 0;
        }
        printf("VALID %d %d %d 0 0 TILE_DRAWN\n", own_open, turn, open_double);
        return 0;
    }

    if (strcmp(action, "PLAY") != 0) {
        printf("INVALID %d %d %d 0 0 ERR_INVALID_ACTION\n", own_open, turn, open_double);
        return 0;
    }

    if (hand_count <= 0) {
        printf("INVALID %d %d %d 0 0 ERR_EMPTY_HAND\n", own_open, turn, open_double);
        return 0;
    }

    if (target != -1 && target != 0 && target != 1) {
        printf("INVALID %d %d %d 0 0 ERR_INVALID_TRAIN\n", own_open, turn, open_double);
        return 0;
    }

    int target_open = mex_open;
    if (target == turn) {
        target_open = own_open;
    } else if (target == opp) {
        target_open = opp_open;
    }

    /* MT-INV-4: Mandatory open-double satisfaction */
    if (open_double != -2 && target != open_double) {
        printf("INVALID %d %d %d 0 0 ERR_MUST_SATISFY_DOUBLE\n", target_open, turn, open_double);
        return 0;
    }

    /* MT-INV-3: Train access rule when no open double is pending */
    if (open_double == -2 && target == opp && !opp_public) {
        printf("INVALID %d %d %d 0 0 ERR_TRAIN_NOT_PUBLIC\n", target_open, turn, open_double);
        return 0;
    }

    /* MT-INV-2: Legal pip matching & orientation */
    if (tile_l < 0 || tile_r < 0 || tile_l > 12 || tile_r > 12) {
        printf("INVALID %d %d %d 0 0 ERR_INVALID_TILE\n", target_open, turn, open_double);
        return 0;
    }
    int new_open = -1;
    if (tile_l == target_open) {
        new_open = tile_r;
    } else if (tile_r == target_open) {
        new_open = tile_l;
    } else {
        printf("INVALID %d %d %d 0 0 ERR_PIP_MISMATCH\n", target_open, turn, open_double);
        return 0;
    }

    /* MT-INV-5: Emptying hand immediately wins the hand (even on a double) */
    if (hand_count == 1) {
        printf("VALID %d %d -2 0 1 HAND_WON\n", new_open, turn);
        return 0;
    }

    /* Double played -> sets open double & grants immediate bonus turn to same player */
    if (tile_l == tile_r) {
        printf("VALID %d %d %d 0 0 DOUBLE_PLAYED_BONUS_TURN\n", new_open, turn, target);
        return 0;
    }

    /* Non-double covered an open double -> clears obligation and advances turn */
    if (open_double != -2) {
        printf("VALID %d %d -2 0 0 DOUBLE_SATISFIED\n", new_open, opp);
        return 0;
    }

    printf("VALID %d %d -2 0 0 MOVE_ACCEPTED\n", new_open, opp);
    return 0;
}
