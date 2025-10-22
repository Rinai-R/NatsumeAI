package logic

import (
	"context"
	"strconv"
	"strings"

	"NatsumeAI/app/common/consts/errno"
	productmodel "NatsumeAI/app/dal/product"
	inventorypb "NatsumeAI/app/services/inventory/inventory"
	"NatsumeAI/app/services/product/internal/svc"
	"NatsumeAI/app/services/product/product"

	"github.com/zeromicro/go-zero/core/logx"
)

type CreateProductLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewCreateProductLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateProductLogic {
	return &CreateProductLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 创建商品
func (l *CreateProductLogic) CreateProduct(in *product.CreateProductReq) (*product.CreateProductResp, error) {
	resp := &product.CreateProductResp{
		StatusCode: errno.InternalError,
		StatusMsg:  "internal error",
	}

	if in == nil {
		resp.StatusCode = errno.InvalidParam
		resp.StatusMsg = "request is nil"
		return resp, nil
	}

	name := strings.TrimSpace(in.GetName())
	description := strings.TrimSpace(in.GetDescription())
	picture := strings.TrimSpace(in.GetPicture())
	price := in.GetPrice()
	merchantID := in.GetMerchantId()

	if name == "" || description == "" || picture == "" || price <= 0 || merchantID <= 0 {
		resp.StatusCode = errno.InvalidParam
		resp.StatusMsg = "invalid product payload"
		return resp, nil
	}

	record := &productmodel.Products{
		MerchantId:  merchantID,
		Name:        name,
		Description: description,
		Picture:     picture,
		Price:       price,
	}

	result, err := l.svcCtx.ProductModel.Insert(l.ctx, record)
	if err != nil {
		l.Logger.Errorf("create product insert failed: %v", err)
		return resp, err
	}

	productID, err := result.LastInsertId()
	if err != nil {
		l.Logger.Errorf("create product get last insert id failed: %v", err)
		return resp, err
	}

	cleanup := func() {
		if delCatErr := l.svcCtx.ProductCategoriesModel.DeleteByProductId(l.ctx, productID); delCatErr != nil && delCatErr != productmodel.ErrNotFound {
			l.Logger.Errorf("create product rollback categories delete failed: %v", delCatErr)
		}
		if delErr := l.svcCtx.ProductModel.Delete(l.ctx, productID); delErr != nil {
			l.Logger.Errorf("create product rollback delete failed: %v", delErr)
		}
	}

	categories := sanitizeCategories(in.GetCategories())
	for _, category := range categories {
		_, err := l.svcCtx.ProductCategoriesModel.Insert(l.ctx, &productmodel.ProductCategories{
			ProductId: productID,
			Category:  category,
		})
		if err != nil {
			l.Logger.Errorf("create product insert category failed: %v", err)
			cleanup()
			return resp, err
		}
	}

	inventoryResp, err := l.svcCtx.InventoryRpc.CreateInventory(l.ctx, &inventorypb.CreateInventoryReq{
		ProductId:  productID,
		Inventory:  in.Stock,
		MerchantId: merchantID,
	})
	if err != nil {
		l.Logger.Errorf("create product initialize inventory rpc failed: %v", err)
		cleanup()
		return resp, err
	}

	if inventoryResp.StatusCode != errno.StatusOK {
		l.Logger.Errorf("create product initialize inventory returned code: %d msg: %s", inventoryResp.StatusCode, inventoryResp.StatusMsg)
		cleanup()
		resp.StatusCode = inventoryResp.StatusCode
		resp.StatusMsg = inventoryResp.StatusMsg
		return resp, nil
	}

	if err := l.svcCtx.Bloom.Add([]byte(strconv.Itoa(int(productID)))); err != nil {
		// 仅打日志，不影响流程
		l.Logger.Error("bloomFilter productId add error: ", productID)
	}

	stored, err := l.svcCtx.ProductModel.FindOne(l.ctx, productID)
	if err != nil {
		l.Logger.Errorf("create product fetch freshly inserted failed: %v", err)
		return resp, err
	}

	protoProduct, err := productModelToProto(stored)
	if err != nil {
		l.Logger.Errorf("create product convert proto failed: %v", err)
		return resp, err
	}
	protoProduct.Categories = categories

	if categoryRecords, err := l.svcCtx.ProductCategoriesModel.ListByProductId(l.ctx, productID); err != nil {
		if err != productmodel.ErrNotFound {
			l.Logger.Errorf("create product list categories failed: %v", err)
			return resp, err
		}
	} else {
		protoProduct.Categories = categoriesFromRecords(categoryRecords)
	}

	resp.StatusCode = errno.StatusOK
	resp.StatusMsg = "ok"
	resp.Product = protoProduct

	return resp, nil
}
